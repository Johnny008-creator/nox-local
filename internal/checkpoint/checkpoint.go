// Package checkpoint snapshots the working tree before each turn so /rewind
// can undo whatever any provider changed. It uses git plumbing with a
// private index file: the user's staging area and branches are untouched,
// and snapshots are ordinary (unreferenced) git tree objects.
package checkpoint

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type Point struct {
	Tree   string
	Label  string
	Taken  time.Time
	Change int // files changed since this point (filled by Diff)
}

type Repo struct {
	root  string
	index string
}

// Open returns nil when dir is not inside a git work tree or git is missing.
func Open(dir string) *Repo {
	out, err := git(context.Background(), dir, nil, "rev-parse", "--show-toplevel")
	if err != nil {
		return nil
	}
	root := strings.TrimSpace(out)
	idx := filepath.Join(os.TempDir(), fmt.Sprintf("nox-index-%x", hash(root)))
	return &Repo{root: root, index: idx}
}

func hash(s string) uint32 {
	var h uint32 = 2166136261
	for i := 0; i < len(s); i++ {
		h = (h ^ uint32(s[i])) * 16777619
	}
	return h
}

func git(ctx context.Context, dir string, env []string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), env...)
	var out, errb bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &errb
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(errb.String())
		if msg == "" {
			msg = err.Error()
		}
		return out.String(), errors.New(msg)
	}
	return out.String(), nil
}

func (r *Repo) priv(ctx context.Context, args ...string) (string, error) {
	return git(ctx, r.root, []string{"GIT_INDEX_FILE=" + r.index}, args...)
}

// Snapshot records the whole working tree (tracked + untracked, honouring
// .gitignore) as a tree object.
func (r *Repo) Snapshot(label string) (Point, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if _, err := r.priv(ctx, "add", "-A", "--", "."); err != nil {
		return Point{}, err
	}
	tree, err := r.priv(ctx, "write-tree")
	if err != nil {
		return Point{}, err
	}
	return Point{Tree: strings.TrimSpace(tree), Label: label, Taken: time.Now()}, nil
}

type change struct {
	status, path string
}

func (r *Repo) diff(ctx context.Context, from, to string) ([]change, error) {
	out, err := git(ctx, r.root, nil, "diff", "--name-status", "--no-renames", "-z", from, to)
	if err != nil {
		return nil, err
	}
	parts := strings.Split(strings.TrimRight(out, "\x00"), "\x00")
	var cs []change
	for i := 0; i+1 < len(parts); i += 2 {
		cs = append(cs, change{parts[i], parts[i+1]})
	}
	return cs, nil
}

// Changed counts files that differ between p and the current tree.
func (r *Repo) Changed(p Point) (int, error) {
	now, err := r.Snapshot("now")
	if err != nil {
		return 0, err
	}
	cs, err := r.diff(context.Background(), p.Tree, now.Tree)
	return len(cs), err
}

// ChangedPaths lists the files that differ between two snapshots.
func (r *Repo) ChangedPaths(from, to Point) ([]string, error) {
	cs, err := r.diff(context.Background(), from.Tree, to.Tree)
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(cs))
	for _, c := range cs {
		out = append(out, c.path)
	}
	return out, nil
}

// Has reports whether a snapshot's tree is still in the repository (git gc
// can prune unreferenced objects).
func (r *Repo) Has(p Point) bool {
	if p.Tree == "" {
		return false
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_, err := git(ctx, r.root, nil, "cat-file", "-e", p.Tree)
	return err == nil
}

// Root is the repository's top folder.
func (r *Repo) Root() string { return r.root }

// Commit stages everything and commits it on the current branch (used by
// auto_commit). It returns the short hash, or "" when there was nothing.
func (r *Repo) Commit(msg string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if _, err := git(ctx, r.root, nil, "add", "-A"); err != nil {
		return "", err
	}
	if _, err := git(ctx, r.root, nil, "diff", "--cached", "--quiet"); err == nil {
		return "", nil // exit 0: nothing staged
	}
	if _, err := git(ctx, r.root, nil, "commit", "-q", "-m", msg); err != nil {
		return "", err
	}
	h, err := git(ctx, r.root, nil, "rev-parse", "--short", "HEAD")
	return strings.TrimSpace(h), err
}

// Worktree creates a sibling worktree on a new branch nox/<name>.
func (r *Repo) Worktree(name string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	path := filepath.Join(filepath.Dir(r.root), filepath.Base(r.root)+"-"+name)
	_, err := git(ctx, r.root, nil, "worktree", "add", "-b", "nox/"+name, path)
	return path, err
}

// Restore puts the working tree back to p: files modified or deleted since
// are restored, files created since are removed. It returns the paths it
// touched.
func (r *Repo) Restore(p Point) ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	now, err := r.Snapshot("before-rewind")
	if err != nil {
		return nil, err
	}
	cs, err := r.diff(ctx, p.Tree, now.Tree)
	if err != nil {
		return nil, err
	}
	var restore, touched []string
	for _, c := range cs {
		touched = append(touched, c.path)
		if c.status == "A" { // created after the checkpoint
			if err := os.Remove(filepath.Join(r.root, filepath.FromSlash(c.path))); err != nil && !os.IsNotExist(err) {
				return touched, err
			}
			continue
		}
		restore = append(restore, c.path)
	}
	if len(restore) > 0 {
		if _, err := r.priv(ctx, "read-tree", p.Tree); err != nil {
			return touched, err
		}
		args := append([]string{"checkout-index", "-f", "--"}, restore...)
		if _, err := r.priv(ctx, args...); err != nil {
			return touched, err
		}
	}
	return touched, nil
}

// relPath turns an absolute path into a path inside the repo, refusing
// anything outside it.
func (r *Repo) relPath(path string) (string, error) {
	if !filepath.IsAbs(path) {
		return "", fmt.Errorf("%s: need an absolute path", path)
	}
	rel, err := filepath.Rel(r.root, filepath.Clean(path))
	if err != nil || rel == "." || strings.HasPrefix(rel, "..") {
		return "", fmt.Errorf("%s is outside the repository", path)
	}
	return filepath.ToSlash(rel), nil
}

// FileDiffers reports whether path is different now from its state at p
// (created, deleted or changed since).
func (r *Repo) FileDiffers(p Point, path string) bool {
	rel, err := r.relPath(path)
	if err != nil {
		return false
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	then, errThen := git(ctx, r.root, nil, "cat-file", "-p", p.Tree+":"+rel)
	now, errNow := os.ReadFile(filepath.Join(r.root, filepath.FromSlash(rel)))
	switch {
	case errThen != nil && errNow != nil:
		return false // absent both times
	case errThen != nil || errNow != nil:
		return true
	}
	return then != string(now)
}

// RestoreFile puts one path (absolute, inside the repo) back to its state
// at p, removing it if it did not exist then.
func (r *Repo) RestoreFile(p Point, path string) error {
	rel, err := r.relPath(path)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if _, err := git(ctx, r.root, nil, "cat-file", "-e", p.Tree+":"+rel); err != nil {
		if err := os.Remove(filepath.Join(r.root, filepath.FromSlash(rel))); err != nil && !os.IsNotExist(err) {
			return err
		}
		return nil
	}
	if _, err := r.priv(ctx, "read-tree", p.Tree); err != nil {
		return err
	}
	_, err = r.priv(ctx, "checkout-index", "-f", "--", rel)
	return err
}

// DiffText is the unified diff from p to the current working tree
// (untracked files included), as git prints it.
func (r *Repo) DiffText(p Point, path string) (string, error) {
	now, err := r.Snapshot("diff")
	if err != nil {
		return "", err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	args := []string{"diff", "--no-color", "--no-ext-diff", "-U3", p.Tree, now.Tree}
	if path != "" {
		args = append(args, "--", path)
	}
	return git(ctx, r.root, nil, args...)
}

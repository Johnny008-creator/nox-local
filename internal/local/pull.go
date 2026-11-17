package local

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

// Pull downloads a profile's GGUF file, resuming a partial download.
// Progress is written to w.
func Pull(ctx context.Context, p Profile, w io.Writer) error {
	if p.Installed() {
		fmt.Fprintf(w, "%s is already downloaded (%s)\n", p.ID, p.Path())
		return nil
	}
	if err := os.MkdirAll(ModelDir(), 0o755); err != nil {
		return err
	}
	part := p.Path() + ".part"
	f, err := os.OpenFile(part, os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	have, _ := f.Seek(0, io.SeekEnd)

	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, p.URL(), nil)
	if have > 0 {
		req.Header.Set("Range", fmt.Sprintf("bytes=%d-", have))
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	switch resp.StatusCode {
	case http.StatusOK:
		if have > 0 { // server ignored the range: start over
			if err := f.Truncate(0); err != nil {
				return err
			}
			_, _ = f.Seek(0, io.SeekStart)
			have = 0
		}
	case http.StatusPartialContent:
	case http.StatusRequestedRangeNotSatisfiable:
		resp.Body.Close()
	default:
		return fmt.Errorf("download %s: HTTP %d", p.URL(), resp.StatusCode)
	}
	total := have + resp.ContentLength
	fmt.Fprintf(w, "pulling %s → %s\n", filepath.Base(p.Path()), ModelDir())
	pw := &progress{w: w, done: have, total: total, start: time.Now(), from: have}
	if _, err := io.Copy(io.MultiWriter(f, pw), resp.Body); err != nil {
		fmt.Fprintln(w)
		return fmt.Errorf("download interrupted (%v) · run pull again to resume", err)
	}
	pw.print(true)
	if err := f.Close(); err != nil {
		return err
	}
	return os.Rename(part, p.Path())
}

type progress struct {
	w                 io.Writer
	done, total, from int64
	start, last       time.Time
}

func (p *progress) Write(b []byte) (int, error) {
	p.done += int64(len(b))
	if time.Since(p.last) > 500*time.Millisecond {
		p.print(false)
		p.last = time.Now()
	}
	return len(b), nil
}

func (p *progress) print(final bool) {
	gb := func(n int64) float64 { return float64(n) / 1e9 }
	speed := float64(p.done-p.from) / time.Since(p.start).Seconds() / 1e6
	pct := 0.0
	if p.total > 0 {
		pct = float64(p.done) / float64(p.total) * 100
	}
	fmt.Fprintf(p.w, "\r  %5.1f%%  %.2f / %.2f GB  %.1f MB/s   ", pct, gb(p.done), gb(p.total), speed)
	if final {
		fmt.Fprintln(p.w)
	}
}

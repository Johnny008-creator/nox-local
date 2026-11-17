package tui

import (
	tea "charm.land/bubbletea/v2"
)

// A small vim layer for the prompt: /vim toggles it. Esc enters normal mode
// (esc while NØX works still interrupts); i a I A o enter insert mode.
// Normal mode: h j k l w b e 0 ^ $ x X D C dd cc gg G and Enter sends.

type vimState struct {
	on      bool
	normal  bool
	pending string // "d", "c", "g" awaiting their second key
}

func vk(code rune, mod tea.KeyMod) tea.KeyPressMsg { return tea.KeyPressMsg{Code: code, Mod: mod} }

// feed sends synthetic keys to the textarea.
func (m *Model) feed(keys ...tea.KeyPressMsg) {
	for _, k := range keys {
		m.ta, _ = m.ta.Update(k)
	}
}

var (
	kLeft      = vk(tea.KeyLeft, 0)
	kRight     = vk(tea.KeyRight, 0)
	kUp        = vk(tea.KeyUp, 0)
	kDown      = vk(tea.KeyDown, 0)
	kHome      = vk(tea.KeyHome, 0)
	kEnd       = vk(tea.KeyEnd, 0)
	kWordRight = vk(tea.KeyRight, tea.ModAlt)
	kWordLeft  = vk(tea.KeyLeft, tea.ModAlt)
	kDel       = vk(tea.KeyDelete, 0)
	kBack      = vk(tea.KeyBackspace, 0)
	kKillEnd   = vk('k', tea.ModCtrl)
	kKillStart = vk('u', tea.ModCtrl)
)

func cmdVim(m *Model, _ string) tea.Cmd {
	m.vim.on = !m.vim.on
	m.vim.normal, m.vim.pending = false, ""
	if m.vim.on {
		m.info("vim mode on · esc for normal mode, i/a to insert · /vim turns it off")
	} else {
		m.info("vim mode off")
	}
	return nil
}

// vimKey handles a key in vim mode; handled=false lets normal processing run.
func (m *Model) vimKey(k tea.KeyPressMsg) (tea.Cmd, bool) {
	if !m.vim.on {
		return nil, false
	}
	s := k.String()
	if !m.vim.normal {
		if s == "esc" && !m.busy && m.ta.Value() != "" {
			m.vim.normal = true
			m.feed(kLeft)
			return nil, true
		}
		return nil, false
	}
	if p := m.vim.pending; p != "" {
		m.vim.pending = ""
		switch p + s {
		case "dd":
			m.feed(kHome, kKillEnd)
		case "cc":
			m.feed(kHome, kKillEnd)
			m.vim.normal = false
		case "dw":
			m.feed(kWordRight)
			m.feed(vk('w', tea.ModCtrl)) // delete word before cursor
		case "gg":
			m.ta.MoveToBegin()
		}
		return nil, true
	}
	switch s {
	case "h", "left":
		m.feed(kLeft)
	case "l", "right":
		m.feed(kRight)
	case "j", "down":
		m.feed(kDown)
	case "k", "up":
		m.feed(kUp)
	case "w", "e":
		m.feed(kWordRight)
	case "b":
		m.feed(kWordLeft)
	case "0", "^", "home":
		m.feed(kHome)
	case "$", "end":
		m.feed(kEnd)
	case "x":
		m.feed(kDel)
	case "X":
		m.feed(kBack)
	case "D":
		m.feed(kKillEnd)
	case "C":
		m.feed(kKillEnd)
		m.vim.normal = false
	case "G":
		m.ta.MoveToEnd()
	case "d", "c", "g":
		m.vim.pending = s
	case "i":
		m.vim.normal = false
	case "a":
		m.feed(kRight)
		m.vim.normal = false
	case "I":
		m.feed(kHome)
		m.vim.normal = false
	case "A":
		m.feed(kEnd)
		m.vim.normal = false
	case "o":
		m.feed(kEnd)
		m.ta.InsertString("\n")
		m.vim.normal = false
	case "O":
		m.feed(kHome)
		m.ta.InsertString("\n")
		m.feed(kUp)
		m.vim.normal = false
	case "enter", "esc", "ctrl+c", "shift+tab", "ctrl+t", "ctrl+k", "ctrl+x", "ctrl+f", "pgup", "pgdown":
		if s == "enter" {
			m.vim.normal = false
		}
		return nil, false // fall through to the normal bindings
	}
	return nil, true // swallow everything else in normal mode
}

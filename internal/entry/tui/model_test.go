package tui

import (
	"os"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestNewModelTextareaHasNoCharLimit(t *testing.T) {
	m := NewModel(nil, nil, "")
	if m.textarea.CharLimit != 0 {
		t.Fatalf("textarea CharLimit = %d, want 0 (no limit)", m.textarea.CharLimit)
	}
}

func TestTextareaAcceptsLongPromptPaste(t *testing.T) {
	data, err := os.ReadFile("../../../idea/hornor/01-truyen-than.prompt.md")
	if err != nil {
		t.Fatalf("read fixture prompt: %v", err)
	}

	m := NewModel(nil, nil, "")
	var cmd tea.Cmd
	m.textarea, cmd = m.textarea.Update(tea.KeyMsg{
		Type:  tea.KeyRunes,
		Runes: []rune(string(data)),
	})
	if cmd != nil {
		_ = cmd
	}
	if got, want := m.textarea.Value(), string(data); got != want {
		t.Fatalf("textarea value length = %d, want %d", len(got), len(want))
	}
}

func TestPromptFileCommandRegistered(t *testing.T) {
	spec, ok := commandRegistryInstance().Find("prompt-file")
	if !ok {
		t.Fatal("prompt-file command is not registered")
	}
	if spec.Usage != "/prompt-file <path>" {
		t.Fatalf("usage = %q", spec.Usage)
	}
	if _, ok := commandRegistryInstance().Find("pf"); !ok {
		t.Fatal("prompt-file alias pf is not registered")
	}
}

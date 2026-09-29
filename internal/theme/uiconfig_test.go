package theme

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
	"go.fuzzyporpoise.dev/book/pkg/book"
	"go.fuzzyporpoise.dev/tint"
)

func TestUIConfigStyledError(t *testing.T) {
	t.Run("plain in non-interactive mode", func(t *testing.T) {
		cfg := &UIConfig{Config: &book.Config{Interactive: false}}
		got := cfg.StyledError(errors.New("boom"))
		if got != "boom" {
			t.Errorf("got %q, want %q", got, "boom")
		}
	})

	t.Run("styled in interactive mode", func(t *testing.T) {
		cfg := &UIConfig{Config: &book.Config{Interactive: true}}
		cfg.Theme = tint.NewTheme(nil)
		got := cfg.StyledError(errors.New("boom"))
		if !strings.Contains(got, "HEAVENS TO MURGATROYD!") {
			t.Errorf("styled error missing header: %q", got)
		}
		if !strings.Contains(got, "boom") {
			t.Errorf("styled error missing message: %q", got)
		}
	})
}

func TestUIConfigLoadTheme(t *testing.T) {
	t.Run("non-interactive uses defaults", func(t *testing.T) {
		cfg := &UIConfig{Config: &book.Config{Interactive: false}}
		if err := cfg.LoadTheme(false); err != nil {
			t.Fatalf("LoadTheme: %v", err)
		}
		if cfg.Theme == nil {
			t.Fatal("expected a compiled theme")
		}
	})

	t.Run("missing file falls back to defaults", func(t *testing.T) {
		cfg := &UIConfig{Config: &book.Config{
			Interactive: true,
			ThemeFile:   filepath.Join(t.TempDir(), "theme.json"),
		}}
		if err := cfg.LoadTheme(true); err != nil {
			t.Fatalf("LoadTheme: %v", err)
		}
		if got := cfg.Theme.Color("primary"); got != lipgloss.Color("#FF4081") {
			t.Errorf("expected default primary, got %v", got)
		}
	})

	t.Run("theme file overrides defaults", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "theme.json")
		data := []byte(`{"palette":{"primary":{"color":"#123456"}}}`)
		if err := os.WriteFile(path, data, 0o600); err != nil {
			t.Fatalf("WriteFile: %v", err)
		}
		cfg := &UIConfig{Config: &book.Config{Interactive: true, ThemeFile: path}}
		if err := cfg.LoadTheme(true); err != nil {
			t.Fatalf("LoadTheme: %v", err)
		}
		if got := cfg.Theme.Color("primary"); got != lipgloss.Color("#123456") {
			t.Errorf("expected overridden primary, got %v", got)
		}
	})

	t.Run("invalid theme file errors", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "theme.json")
		if err := os.WriteFile(path, []byte(`{"bogus":true}`), 0o600); err != nil {
			t.Fatalf("WriteFile: %v", err)
		}
		cfg := &UIConfig{Config: &book.Config{Interactive: true, ThemeFile: path}}
		if err := cfg.LoadTheme(true); err == nil {
			t.Fatal("expected an error for an unknown theme key")
		}
	})
}

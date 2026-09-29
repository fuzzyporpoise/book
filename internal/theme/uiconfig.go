// Package theme carries the book CLI's runtime presentation state: the tint
// theme loaded from theme.json, view template overrides, and interactive error
// styling. Storage configuration lives on the embedded *book.Config.
package theme

import (
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"os"

	"go.fuzzyporpoise.dev/book/pkg/book"
	"go.fuzzyporpoise.dev/tint"
)

const errorBullet string = "󰯷" // "nf-md-alpha_e_box_outline

// UIConfig carries runtime presentation state (theme, templates) alongside
// the embedded storage configuration. The CLI and TUI layers work with
// UIConfig; storage calls receive the embedded *book.Config.
type UIConfig struct {
	*book.Config
	Theme     *tint.Theme
	Templates map[string]book.ViewTemplate
}

// LoadTheme loads the theme file over the tint defaults, falling back to the
// built-in theme when the file is absent or a non-interactive run is requested.
func (cfg *UIConfig) LoadTheme(interactive bool) error {
	cfg.Theme = tint.NewTheme(nil)

	// load theme file or use defaults for TUI/interactive features
	if !interactive || cfg.ThemeFile == "" {
		return nil
	}

	raw, err := tint.LoadThemeConfig(cfg.ThemeFile)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("load theme: %w", err)
	}
	cfg.Theme = tint.NewTheme(raw)
	return nil
}

// LoadTemplates loads user template overrides on top of built-in defaults.
func (cfg *UIConfig) LoadTemplates() error {
	cfg.Templates = make(map[string]book.ViewTemplate)

	// Start with defaults
	maps.Copy(cfg.Templates, book.DefaultViewTemplates)

	data, err := os.ReadFile(cfg.TemplateFile)
	if os.IsNotExist(err) {
		// Not an error — user hasn't customized, defaults are fine
		return nil
	}
	if err != nil {
		return fmt.Errorf("read templates: %w", err)
	}

	var userTmpls map[string]book.ViewTemplate
	if err := json.Unmarshal(data, &userTmpls); err != nil {
		return fmt.Errorf("parse templates: %w", err)
	}

	// Overlay user partials onto defaults
	for k, user := range userTmpls {
		base, ok := cfg.Templates[k]
		if !ok {
			// Unknown key — skip or warn
			continue
		}
		if user.PrimaryTitle != "" {
			base.PrimaryTitle = user.PrimaryTitle
		}
		if user.SecondaryTitle != "" {
			base.SecondaryTitle = user.SecondaryTitle
		}
		if user.ListTitle != "" {
			base.ListTitle = user.ListTitle
		}
		cfg.Templates[k] = base
	}

	return nil
}

// StyledError returns a styled error string for interactive mode, or plain text otherwise.
func (cfg *UIConfig) StyledError(e error) string {
	if !cfg.Interactive {
		return e.Error()
	}
	// return styled error only in interactive mode
	return cfg.Theme.Style("highlight").Render("HEAVENS TO MURGATROYD!") + "\n" +
		cfg.Theme.Style("error").Render(errorBullet, e.Error())
}

package internal

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/lipgloss/v2"
	"github.com/charmbracelet/lipgloss/v2/table"
)

const (
	configDirName  = "prism"
	configFileName = "config.json"
)

var configPathOverride = os.Getenv("PRISM_CONFIG_FILE")

type Config struct {
	Verbose   bool `json:"verbose"`
	OnlyFails bool `json:"only_fails"`
	NoBar     bool `json:"no_bar"`
	NoColor   bool `json:"no_color"`
	ShowColor bool `json:"show_color"`
	Watch     bool `json:"watch"`
}

// GlobalConfig holds the active configuration for the current process.
var GlobalConfig Config

// ClearConfig will try and remove any config files or directories created by prism.
func ClearConfig() error {
	if configPathOverride != "" {
		// Override stuff. Just delete the file, can't guarantee if the user put it in an empty dir
		_, err := os.Stat(configPathOverride)
		if err != nil {
			return err
		}

		return os.Remove(configPathOverride)
	}

	path, err := configFilePath()
	if err != nil {
		return err
	}

	_, err = os.Stat(path)
	if err != nil {
		return err
	}

	err = os.Remove(path)
	if err != nil {
		return err
	}

	parentDir := filepath.Dir(path)
	if !strings.HasSuffix(parentDir, "prism") {
		return fmt.Errorf("File removed, but containing directory not named `prism`, so not attempting to remove it")
	}

	return os.Remove(parentDir)
}

// LoadConfig reads the persisted configuration file, if it exists.
func LoadConfig() (Config, error) {
	cfg := Config{}

	path, err := configFilePath()
	if err != nil {
		return cfg, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return cfg, nil
		}
		return cfg, fmt.Errorf("Read config file: %w", err)
	}

	if len(data) == 0 {
		return cfg, nil
	}

	if err := json.Unmarshal(data, &cfg); err != nil {
		return cfg, fmt.Errorf("Parse config file: %w", err)
	}

	return cfg, nil
}

// SaveConfig writes the provided configuration to disk, persisting supported settings.
func SaveConfig(cfg Config) error {
	path, err := configFilePath()
	if err != nil {
		return err
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("Create config directory: %w", err)
	}

	payload, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("Encode config: %w", err)
	}
	payload = append(payload, '\n')

	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, payload, 0o644); err != nil {
		return fmt.Errorf("Write config temp file: %w", err)
	}

	if err := os.Rename(tmpPath, path); err != nil {
		if removeErr := os.Remove(path); removeErr != nil && !errors.Is(removeErr, fs.ErrNotExist) {
			_ = os.Remove(tmpPath)
			return fmt.Errorf("Replace config file: %w", err)
		}
		if err := os.Rename(tmpPath, path); err != nil {
			_ = os.Remove(tmpPath)
			return fmt.Errorf("Replace config file: %w", err)
		}
	}

	return nil
}

func SetConfig(cfg Config, key string, value bool) error {
	var outBool string
	switch key {
	case "only_fails", "only-fails":
		cfg.OnlyFails = value
		outBool = styleBool(cfg.OnlyFails)
	case "verbose":
		cfg.Verbose = value
		outBool = styleBool(cfg.Verbose)
	case "no_bar", "no-bar":
		cfg.NoBar = value
		outBool = styleBool(cfg.NoBar)
	case "no_color", "no-color":
		cfg.NoColor = value
		outBool = styleBool(cfg.NoColor)

		cfg.ShowColor = !value
		fmt.Printf("\n%v -> %v", "show_color", styleBool(cfg.ShowColor))
	case "show_color", "show-color":
		cfg.ShowColor = value
		outBool = styleBool(cfg.ShowColor)

		cfg.NoColor = !value
		fmt.Printf("\n%v -> %v", "no_color", styleBool(cfg.NoColor))
	default:
		return fmt.Errorf("Unknown configuration key %q", key)
	}
	if err := SaveConfig(cfg); err != nil {
		return fmt.Errorf("Error saving config: %w", err)
	}

	fmt.Printf("\n%v -> %v\n\n", key, outBool)
	return nil
}

func PrintConfig(cfg Config) {
	table := table.New().
		Rows(
			[]string{"no_bar", styleBool(cfg.NoBar)},
			[]string{"no_color", styleBool(cfg.NoColor)},
			[]string{"show_color", styleBool(cfg.ShowColor)},
			[]string{"only_fails", styleBool(cfg.OnlyFails)},
			[]string{"verbose", styleBool(cfg.Verbose)},
		).
		Border(lipgloss.HiddenBorder())

	fmt.Println(table.String())
}

func configFilePath() (string, error) {
	if configPathOverride != "" {
		return configPathOverride, nil
	}

	dir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("Resolve user config dir: %w", err)
	}

	return filepath.Join(dir, configDirName, configFileName), nil
}

func styleBool(in bool) string {
	style := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.BrightGreen)
	if !in {
		style = style.Foreground(lipgloss.BrightRed)
	}

	if GlobalConfig.NoColor {
		style = style.Foreground(lipgloss.NoColor{})
	}

	return style.Render(strings.ToTitle(fmt.Sprintf("%t", in)))
}

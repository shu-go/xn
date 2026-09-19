package main

import (
	"bytes"

	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

type config struct {
	Gmail struct {
		ClientID     string `toml:"ClientID,omitempty"`
		ClientSecret string `toml:"ClientSecret,omitempty"`
		Token        string `toml:"Token,omitempty"`
	}
	Slack struct {
		ClientID     string `toml:"ClientID,omitempty"`
		ClientSecret string `toml:"ClientSecret,omitempty"`
		AccessToken  string `toml:"AccessToken,omitempty"`
	}
	Pushbullet struct {
		ClientID     string `toml:"ClientID,omitempty"`
		ClientSecret string `toml:"ClientSecret,omitempty"`
		AccessToken  string `toml:"AccessToken,omitempty"`
	}
	Teams struct {
		WebhookURL string `toml:"WebhookURL,omitempty"`
	}
	Discord struct {
		WebhookURL string `toml:"WebhookURL,omitempty"`
	}
	Mailgun struct {
		PublicAPIKey  string `toml:"PublicAPIKey,omitempty"`
		PrivateAPIKey string `toml:"PrivateAPIKey,omitempty"`
		Domain        string `toml:"Domain,omitempty"`
	}
}

const configFileName string = "xn.conf"

func determineConfigPath(defaultValue string) string {
	if defaultValue != "" {
		return defaultValue
	}

	// wd
	wdConfigPath := filepath.Join(".", configFileName)
	if _, err := os.Stat(wdConfigPath); err == nil {
		return wdConfigPath
	}

	// exe
	if exepath, err := os.Executable(); err == nil {
		exeConfigPath := filepath.Join(filepath.Dir(exepath), "xn.conf")
		if _, err := os.Stat(exeConfigPath); err == nil {
			return exeConfigPath
		}
	}

	return wdConfigPath
}

func loadConfig(filePath string) (*config, error) {
	filePath = determineConfigPath(filePath)

	cfg := &config{}
	_, err := toml.DecodeFile(filePath, cfg)
	if err != nil {
		cfg = &config{}
	}

	return cfg, nil
}

func saveConfig(config *config, filePath string) error {
	filePath = determineConfigPath(filePath)

	buf := new(bytes.Buffer)
	if err := toml.NewEncoder(buf).Encode(config); err != nil {
		return err
	}
	return os.WriteFile(filePath, buf.Bytes(), 0600)
}

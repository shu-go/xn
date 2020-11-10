package main

import (
	"bytes"
	"fmt"
	"github.com/BurntSushi/toml"
	"io/ioutil"
	"os"
)

type config struct {
	Gmail struct {
		ClientID     string `toml:"ClientID,omitempty"`
		ClientSecret string `toml:"ClientSecret,omitempty"`
		RefreshToken string `toml:"RefreshToken,omitempty"`
		User         string `toml:"User,omitempty"`
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
	Mailgun struct {
		PublicAPIKey  string `toml:"PublicAPIKey,omitempty"`
		PrivateAPIKey string `toml:"PrivateAPIKey,omitempty"`
		Domain        string `toml:"Domain,omitempty"`
	}
}

func loadConfig(fileName string) (*config, error) {
	config := &config{}
	_, err := toml.DecodeFile(fileName, config)
	if err != nil {
		fmt.Fprintf(os.Stderr, "missing %v. -> creating with minimal contents...", fileName)
		if err := saveConfig(config, fileName); err != nil {
			return config, fmt.Errorf("failed to access to config: %v", err)
		}
		fmt.Fprintf(os.Stderr, "created.\n")
	}

	return config, nil
}

func saveConfig(config *config, fileName string) error {
	buf := new(bytes.Buffer)
	if err := toml.NewEncoder(buf).Encode(config); err != nil {
		return err
	}
	return ioutil.WriteFile(fileName, buf.Bytes(), 0700)
}

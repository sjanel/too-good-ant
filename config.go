package main

import (
	"encoding/json"
	"fmt"
	"os"
)

type Config struct {
	AccountEmail             string       `json:"accountEmail"`
	Language                 string       `json:"language"`
	MinRequestsPeriodSeconds int          `json:"minRequestsPeriodSeconds"`
	SearchConfig             SearchConfig `json:"searchConfig"`
	SendConfig               SendConfig   `json:"sendConfig"`
	Verbose                  bool         `json:"verbose"`
}

type SearchConfig struct {
	Origin        Location `json:"origin"`
	RadiusInKm    int      `json:"radiusInKm"`
	FavoritesOnly bool     `json:"favoritesOnly"`
	WithStockOnly bool     `json:"withStockOnly"`
}

const (
	kNoSend    string = ""
	kSendEmail        = "email"
)

type SendConfig struct {
	EmailConfig EmailConfig `json:"emailConfig"`
	SendAction  string      `json:"sendAction"`
}

type EmailConfig struct {
	EmailFrom         string `json:"emailFrom"`
	EmailTo           string `json:"emailTo"`
	GmailApiKeyFile   string `json:"gmailApiKeyFile"`
	OauthPortCallback int    `json:"oauthPortCallBack"`
}

func ReadConfigFromFile(filePath string) (*Config, error) {
	configDataBytes, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("error from os.ReadFile: %w", err)
	}

	config := &Config{}

	err = json.Unmarshal(configDataBytes, config)
	if err != nil {
		return nil, fmt.Errorf("error from json.Unmarshal: %w", err)
	}

	glog.Printf("loaded configuration from %v\n", filePath)

	return config, err
}

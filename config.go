package main

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

type Config struct {
	TooGoodToGoConfig TooGoodToGoConfig `json:"tooGoodToGoConfig"`
	SendConfig        SendConfig        `json:"sendConfig"`
	Verbose           bool              `json:"verbose"`
}

// Custom duration to be able to unmarshall it from strings
type Duration struct {
	time.Duration
}

type TooGoodToGoConfig struct {
	AccountEmail               string       `json:"accountEmail"`
	Language                   string       `json:"language"`
	MinRequestsPeriod          Duration     `json:"minRequestsPeriod"`
	ActiveOrdersReminderPeriod Duration     `json:"activeOrdersReminderPeriod"`
	TokenValidityDuration      Duration     `json:"tokenValidityDuration"`
	SearchConfig               SearchConfig `json:"searchConfig"`
	UseGzipEncoding            bool         `json:"useGzipEncoding"`
}

type Location struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
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

func (duration *Duration) UnmarshalJSON(b []byte) error {
	var unmarshalledJson interface{}

	err := json.Unmarshal(b, &unmarshalledJson)
	if err != nil {
		return fmt.Errorf("error from json.Unmarshal: %w", err)
	}

	switch value := unmarshalledJson.(type) {
	case string:
		duration.Duration, err = time.ParseDuration(value)
		if err != nil {
			return fmt.Errorf("error from time.ParseDuration: %w", err)
		}
	default:
		return fmt.Errorf("invalid duration: %#v, provide it as string", unmarshalledJson)
	}

	return nil
}

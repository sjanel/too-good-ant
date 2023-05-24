package main

import (
	"testing"
	"time"
)

const (
	kExampleConfigPath = "data/example_config.json"
)

func TestLoadConfig(t *testing.T) {
	config, err := ReadConfigFromFile(kExampleConfigPath)
	if err != nil {
		t.Fatalf("error from ReadConfigFromFile: %v", err)
	}

	expectedConfig := Config{
		TooGoodToGoConfig: TooGoodToGoConfig{
			AccountEmail: "myemail@email.com",
			Language:     "en-UK; fr-FR",
			MinRequestsPeriod: Duration{
				Duration: time.Duration(30) * time.Second,
			},
			ActiveOrdersReminderPeriod: Duration{
				Duration: time.Duration(10) * time.Minute,
			},
			SearchConfig: SearchConfig{
				Origin: Location{
					Latitude:  41.902782,
					Longitude: 12.496366,
				},
				RadiusInKm:    3,
				FavoritesOnly: true,
				WithStockOnly: true,
			},
			UseGzipEncoding: true,
		},
		SendConfig: SendConfig{
			EmailConfig: EmailConfig{
				EmailFrom:         "emailfrom@email.com",
				EmailTo:           "emailto1@email.com,emailto2@email.com",
				GmailApiKeyFile:   "secrets/client_secret_123456.apps.googleusercontent.com.json",
				OauthPortCallback: 10010,
			},
			SendAction: "email",
		},
		Verbose: false,
	}

	if expectedConfig != *config {
		t.Fatalf("expected config %v, got %v", expectedConfig, config)
	}
}

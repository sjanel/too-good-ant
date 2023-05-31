package main

import (
	"reflect"
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
			Accounts: []TooGoodToGoAccount{
				{
					Email:     "myemail1@email.com",
					UserAgent: "<MyUserAgent1-OrEmpty>",
				},
				{
					Email:     "myemail2@email.com",
					UserAgent: "<MyUserAgent2-OrEmpty>",
				},
			},
			Language: "en-UK; fr-FR",
			AverageRequestsPeriod: Duration{
				Duration: time.Duration(45) * time.Second,
			},
			TooManyRequestsPausePeriod: Duration{
				Duration: time.Duration(1)*time.Hour + time.Duration(30)*time.Minute,
			},
			ActiveOrdersReminderPeriod: Duration{
				Duration: time.Duration(10) * time.Minute,
			},
			TokenValidityDuration: Duration{
				Duration: time.Duration(48) * time.Hour,
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
			WhatsAppConfig: WhatsAppConfig{
				GroupNameTo: "My WhatsApp Group Name",
				UserNameTo:  "My WhatsApp User Name",
			},
			SendAction: SendEmail,
		},
		Verbose: false,
	}

	if !reflect.DeepEqual(expectedConfig, *config) {
		t.Fatalf("expected config %v, got %v", expectedConfig, config)
	}
}

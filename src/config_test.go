package tga

import (
	"reflect"
	"testing"
	"time"
)

const (
	kExampleConfigPath = "testdata/example_config.json"
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
			LogInEmailValidationRequestsPeriod: Duration{
				Duration: time.Duration(15) * time.Second,
			},
			LogInEmailValidationTimeoutDuration: Duration{
				Duration: time.Duration(30) * time.Minute,
			},
			LogInValidityDuration: Duration{
				Duration: time.Duration(48) * time.Hour,
			},
			TokenValidityDuration: Duration{
				Duration: time.Duration(8) * time.Hour,
			},
			SearchConfig: SearchConfig{
				Origin: Location{
					Latitude:  41.902782,
					Longitude: 12.496366,
				},
				RadiusInKm:    3,
				NbMaxResults:  20,
				FavoritesOnly: true,
				WithStockOnly: true,
			},
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

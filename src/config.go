package tga

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

type TooGoodToGoAccount struct {
	Email     string `json:"email"`
	UserAgent string `json:"userAgent"`
}

type TooGoodToGoConfig struct {
	Accounts                            []TooGoodToGoAccount `json:"accounts"`
	Language                            string               `json:"language"`
	AverageRequestsPeriod               Duration             `json:"averageRequestsPeriod"`
	TooManyRequestsPausePeriod          Duration             `json:"tooManyRequestsPausePeriod"`
	ActiveOrdersReminderPeriod          Duration             `json:"activeOrdersReminderPeriod"`
	LogInEmailValidationTimeoutDuration Duration             `json:"logInEmailValidationTimeoutDuration"`
	LogInValidityDuration               Duration             `json:"logInValidityDuration"`
	TokenValidityDuration               Duration             `json:"tokenValidityDuration"`
	SearchConfig                        SearchConfig         `json:"searchConfig"`
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

type SendActionType int

const (
	NoSend SendActionType = iota
	SendEmail
	SendWhatsApp
)

func (s SendActionType) String() string {
	switch s {
	case NoSend:
		return ""
	case SendEmail:
		return "email"
	case SendWhatsApp:
		return "whatsapp"
	}
	return "<error>"
}

func NewSendActionType(str string) (SendActionType, error) {
	if str == "" {
		return NoSend, nil
	}
	if str == "email" {
		return SendEmail, nil
	}
	if str == "whatsapp" {
		return SendWhatsApp, nil
	}
	return -1, fmt.Errorf("unknown send action type %v", str)
}

func (s SendActionType) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.String())
}

func (s *SendActionType) UnmarshalJSON(b []byte) error {
	sendAction, err := NewSendActionType(string(b[1 : len(b)-1]))
	if err != nil {
		return fmt.Errorf("error from NewSendActionType: %w", err)
	}
	*s = sendAction
	return nil
}

type SendConfig struct {
	EmailConfig    EmailConfig    `json:"emailConfig"`
	WhatsAppConfig WhatsAppConfig `json:"whatsAppConfig"`
	SendAction     SendActionType `json:"sendAction"`
}

type EmailConfig struct {
	EmailFrom         string `json:"emailFrom"`
	EmailTo           string `json:"emailTo"`
	GmailApiKeyFile   string `json:"gmailApiKeyFile"`
	OauthPortCallback int    `json:"oauthPortCallBack"`
}

type WhatsAppConfig struct {
	GroupNameTo string `json:"groupNameTo"`
	UserNameTo  string `json:"userNameTo"`
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

	if len(config.TooGoodToGoConfig.Accounts) == 0 {
		return nil, fmt.Errorf("you need to specify at least one too good to go account\n")
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

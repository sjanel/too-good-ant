package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/gmail/v1"
)

const (
	kStateToken = "too-good-to-go-state-token-redirect-url"
)

func SetupConfig(emailConfig EmailConfig) (*oauth2.Config, error) {
	// Reads in our credentials
	secret, err := os.ReadFile(emailConfig.GmailApiKeyFile)
	if err != nil {
		return nil, fmt.Errorf("error from os.ReadFile: %w", err)
	}

	// Creates a oauth2.Config using the secret
	// The second parameter is the scope, in this case we only want to send email
	config, err := google.ConfigFromJSON(secret, gmail.GmailSendScope)
	if err != nil {
		return config, fmt.Errorf("error from google.ConfigFromJSON: %w", err)
	}

	config.RedirectURL = redirectUrl(emailConfig.OauthPortCallback)

	return config, nil
}

func NewGmailClient(emailConfig EmailConfig) (*gmail.Service, error) {
	config, err := SetupConfig(emailConfig)
	if err != nil {
		return nil, fmt.Errorf("error from SetupConfig: %w", err)
	}

	// Creates a URL for the user to follow
	url := config.AuthCodeURL(kStateToken, oauth2.AccessTypeOffline)

	codeChan := make(chan string)

	go ListenToGoogleRedirect(emailConfig.OauthPortCallback, codeChan)

	err = OpenBrowser(url)
	if err != nil {
		return nil, fmt.Errorf("error from OpenBrowser: %w", err)
	}

	// Grabs the authorization code from the web page through the channel provided
	code := <-codeChan

	// Exchange the auth code for an access token
	tok, err := config.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("error from conf.Exchange: %w", err)
	}

	// Create the *http.Client using the access token
	client := config.Client(ctx, tok)

	// Create a new gmail service using the client
	gmailService, err := gmail.New(client)
	if err != nil {
		return nil, fmt.Errorf("error from gmail.New: %w", err)
	}

	glog.Printf("gmail service successfully authenticated\n")

	return gmailService, nil
}

func AuthCodeValidationCallBack(srv *http.Server, codeChan chan<- string) http.HandlerFunc {
	return func(res http.ResponseWriter, req *http.Request) {
		// Check that state matches
		states, hasState := req.URL.Query()["state"]
		if !hasState || len(states) == 0 || states[0] != kStateToken {
			res.WriteHeader(http.StatusBadRequest)
			res.Write([]byte(fmt.Sprintf("unexpected request %v\n", *req)))
			return
		}

		codes, hasCode := req.URL.Query()["code"]
		if !hasCode || len(codes) == 0 {
			res.WriteHeader(http.StatusBadRequest)
			res.Write([]byte(fmt.Sprintf("unexpected request %v\n", *req)))
			return
		}

		code := codes[0]

		res.Write([]byte(fmt.Sprintf("successfully validated code, you can come back to the command line application and close this window\n")))
		codeChan <- code

		go srv.Shutdown(context.Background())
	}
}

func ListenToGoogleRedirect(redirectUrlPort int, codeChan chan<- string) {
	mux := http.NewServeMux()

	srv := http.Server{
		Addr:    fmt.Sprintf("localhost:%v", redirectUrlPort),
		Handler: mux,
	}

	mux.HandleFunc("/", AuthCodeValidationCallBack(&srv, codeChan))

	// run server
	glog.Printf("started server on callback URL for authentication validation: '%v'\n", redirectUrl(redirectUrlPort))
	if err := srv.ListenAndServe(); err != http.ErrServerClosed {
		log.Fatal(err)
	}
	glog.Printf("stopped server on callback URL\n")
}

func redirectUrl(port int) string {
	return fmt.Sprintf("http://localhost:%v/callback", port)
}

package main

import (
	"context"
	"flag"
	"log"
	"math/rand"
	"os"
	"reflect"
	"time"

	"google.golang.org/api/gmail/v1"
)

var (
	ctx  = context.Background()
	glog = log.Default()
)

func main() {
	verbose := flag.Bool("v", false, "Trace requests information for debugging")
	quiet := flag.Bool("q", false, "Quiet: force verbose deactivation")

	flag.Parse()

	const kConfigFilePath = "secrets/config.json"

	config, err := ReadConfigFromFile(kConfigFilePath)
	if os.IsNotExist(err) {
		glog.Fatalf("you need to create file %v that will be loaded and used as your personal configuration", kConfigFilePath)
	} else if err != nil {
		glog.Fatalf("error from ReadConfigFromFile: %v", err)
	}

	if *quiet {
		config.Verbose = false
	} else if *verbose {
		config.Verbose = true
	}

	glog.Printf("starting too good to go service for %v\n", config.AccountEmail)

	tooGoodToGoClient := NewTooGooToGoClient(config.AccountEmail, config.Language, config.Verbose)

	var gmailService *gmail.Service
	if config.SendConfig.SendAction == "email" {
		// Create a new gmail service using the client
		gmailService, err = CreateGmailService(config.SendConfig.EmailConfig)
		if err != nil {
			glog.Fatalf("error from CreateGmailService: %v", err)
		}
	}

	var lastStores []Store

	for {
		stores, err := tooGoodToGoClient.ListStores(config.SearchConfig)
		if err != nil {
			glog.Fatalf("error from ListFavoriteStores: %v", err)
		}

		if gmailService != nil && len(stores) > 0 && !reflect.DeepEqual(lastStores, stores) {
			SendStoresByEmail(gmailService, config.SendConfig.EmailConfig, stores)
			lastStores = stores
		}

		timeSleepSeconds := config.MinRequestsPeriodSeconds + rand.Intn(config.MinRequestsPeriodSeconds)
		time.Sleep(time.Duration(timeSleepSeconds) * time.Second)
	}
}

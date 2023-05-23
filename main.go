package main

import (
	"context"
	"flag"
	"log"
	"math/rand"
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

	flag.Parse()

	config, err := ReadConfigFromFile("secrets/config.json")
	if err != nil {
		glog.Fatalf("error from ReadConfigFromFile: %v", err)
	}

	if *verbose {
		config.Verbose = true
	}

	glog.Printf("starting too good to go service for %v\n", config.AccountEmail)

	lastApkVersion, err := GetLastApkVersion()
	if err != nil {
		glog.Fatalf("error from GetLastApkVersion: %v", err)
	}

	tooGoodToGoClient := NewTooGooToGoClient(config.AccountEmail, lastApkVersion, config.Language, config.Verbose)

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
		}

		lastStores = stores

		timeSleepSeconds := config.MinRequestsPeriodSeconds + rand.Intn(config.MinRequestsPeriodSeconds)
		time.Sleep(time.Duration(timeSleepSeconds) * time.Second)
	}
}

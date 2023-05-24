package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"reflect"
	"syscall"

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

	// Capture SIGTERM for graceful shutdown
	stopSignalReceived := false
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt)
	signal.Notify(signalChan, syscall.SIGTERM)
	go func() {
		for sig := range signalChan {
			glog.Printf("%v signal received\n", sig)
			stopSignalReceived = true
		}
	}()

	glog.Printf("starting too good to go ant for %v\n", config.TooGoodToGoConfig.AccountEmail)

	tooGoodToGoClient := NewTooGooToGoClient(&config.TooGoodToGoConfig, config.Verbose)

	var gmailService *gmail.Service
	if config.SendConfig.SendAction == "email" {
		// Create a new gmail service using the client
		gmailService, err = CreateGmailService(config.SendConfig.EmailConfig)
		if err != nil {
			glog.Fatalf("error from CreateGmailService: %v", err)
		}
	}

	var lastStoresSent []Store

	for !stopSignalReceived {
		stores, err := tooGoodToGoClient.ListStores()
		if err != nil {
			glog.Fatalf("error from ListStores: %v", err)
		}

		_, err = tooGoodToGoClient.ListOpenedOrders()
		if err != nil {
			glog.Fatalf("error from ListOpenedOrders: %v", err)
		}

		if gmailService != nil {
			if len(stores) > 0 && !reflect.DeepEqual(lastStoresSent, stores) {
				SendStoresByEmail(gmailService, config.SendConfig.EmailConfig, stores)
				lastStoresSent = stores
			}
		}

	}

	err = tooGoodToGoClient.writeAuthorizationDataToFile()
	if err != nil {
		glog.Printf("error in tooGoodToGoClient.WriteAuthorizationDataToFile: %v\n", err)
		err = nil
	}

	glog.Printf("bye\n")
}

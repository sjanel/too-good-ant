package main

import (
	"bytes"
	"context"
	"flag"
	"log"
	"os"
	"reflect"
)

var (
	ctx  = context.Background()
	glog = log.Default()
)

func main() {
	forceVerbose := flag.Bool("v", false, "Trace requests information for debugging")
	forceQuiet := flag.Bool("q", false, "Quiet: force verbose deactivation")
	configFilePath := flag.String("conf", "secrets/config.json", "Configuration file path")

	flag.Parse()

	config, err := ReadConfigFromFile(*configFilePath)
	if os.IsNotExist(err) {
		glog.Fatalf("you need to create file %v that will be loaded and used as your personal configuration", *configFilePath)
	} else if err != nil {
		glog.Fatalf("error from ReadConfigFromFile: %v", err)
	}

	if *forceVerbose {
		config.Verbose = true
	} else if *forceQuiet {
		config.Verbose = false
	}

	sender, err := NewSender(config.SendConfig)
	if err != nil {
		glog.Fatalf("error from NewSender: %v", err)
	}
	defer sender.Close()

	glog.Printf("starting too good to go ant for %v\n", config.TooGoodToGoConfig.AccountEmail)

	tooGoodToGoClient := NewTooGooToGoClient(&config.TooGoodToGoConfig, config.Verbose)

	// Capture SIGTERM for graceful shutdown
	stopSignalReceived := false
	GracefulShutdownHook(&stopSignalReceived)

	var lastStoresSent []Store

	_, err = tooGoodToGoClient.PaymentMethods(Adyen)
	if err != nil {
		glog.Fatalf("error from PaymentMethods: %v", err)
	}

	for !stopSignalReceived {
		stores, err := tooGoodToGoClient.ListStores()
		if err != nil {
			glog.Fatalf("error from ListStores: %v", err)
		}

		_, err = tooGoodToGoClient.ListOpenedOrders()
		if err != nil {
			glog.Fatalf("error from ListOpenedOrders: %v", err)
		}

		if len(stores) > 0 && !reflect.DeepEqual(lastStoresSent, stores) {
			storeMessage := bytes.NewBuffer([]byte{})
			for _, store := range stores {
				storeMessage.WriteString(store.String())
				storeMessage.WriteByte('\n')
			}
			_, err = sender.Write(storeMessage.Bytes())
			if err != nil {
				glog.Fatalf("error from sender.Write: %v", err)
			}
			lastStoresSent = stores
		}

	}

	err = tooGoodToGoClient.writeAuthorizationDataToFile()
	if err != nil {
		glog.Printf("error in tooGoodToGoClient.WriteAuthorizationDataToFile: %v\n", err)
		err = nil
	}

	glog.Printf("bye\n")
}

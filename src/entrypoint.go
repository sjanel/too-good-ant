package tga

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"reflect"
)

var (
	ctx  = context.Background()
	glog = log.Default()
)

func Start() {
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

	glog.Printf("starting too good to go ant for %v accounts\n", len(config.TooGoodToGoConfig.Accounts))

	tooGoodToGoClient := NewTooGooToGoClient(&config.TooGoodToGoConfig, config.Verbose)
	defer tooGoodToGoClient.Close()

	// Capture SIGTERM for graceful shutdown
	stopSignalReceived := false
	GracefulShutdownHook(&stopSignalReceived)

	var lastStoresSent []Store

	for !stopSignalReceived {
		stores, err := tooGoodToGoClient.ListStores()
		if err != nil {
			glog.Fatalf("error from ListStores: %v", err)
		}

		if len(stores) > 0 && !reflect.DeepEqual(lastStoresSent, stores) {
			storeMessage, err := computeStoresMessage(stores)
			if err != nil {
				glog.Printf("error from computeStoresMessage: %v", err)
			}
			_, err = sender.Write(storeMessage)
			if err != nil {
				glog.Printf("error from sender.Write: %v", err)
			}
			lastStoresSent = stores
		}

		_, err = tooGoodToGoClient.ListOpenedOrders()
		if err != nil {
			glog.Fatalf("error from ListOpenedOrders: %v", err)
		}
	}

	glog.Printf("exiting too good ant\n")
}

func computeStoresMessage(stores []Store) ([]byte, error) {
	storeMessage := bytes.NewBuffer([]byte{})
	for _, store := range stores {
		_, err := storeMessage.WriteString(store.String())
		if err != nil {
			return nil, fmt.Errorf("error from storeMessage.WriteString: %w", err)
		}
		err = storeMessage.WriteByte('\n')
		if err != nil {
			return nil, fmt.Errorf("error from storeMessage.WriteByte: %w", err)
		}
	}
	return storeMessage.Bytes(), nil
}

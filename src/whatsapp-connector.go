package tga

import (
	"context"
	"fmt"
	"os"

	_ "github.com/mattn/go-sqlite3"
	"github.com/mdp/qrterminal"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/store/sqlstore"
	waLog "go.mau.fi/whatsmeow/util/log"
)

func NewWhatsAppClient(whatsAppConfig WhatsAppConfig) (*whatsmeow.Client, error) {
	const kWhatsAppAuthFilePath = "secrets/whatsapp.db"

	container, err := sqlstore.New("sqlite3", fmt.Sprintf("file:%v?_foreign_keys=on", kWhatsAppAuthFilePath), waLog.Noop)
	if err != nil {
		return nil, fmt.Errorf("error from sqlstore.New: %w", err)
	}
	deviceStore, err := container.GetFirstDevice()
	if err != nil {
		return nil, fmt.Errorf("error from container.GetFirstDevice: %w", err)
	}
	client := whatsmeow.NewClient(deviceStore, waLog.Noop)
	if client.Store.ID == nil {
		// No ID stored, new login
		qrChan, _ := client.GetQRChannel(context.Background())
		err = client.Connect()
		if err != nil {
			return nil, fmt.Errorf("error from client.Connect: %w", err)
		}
		for evt := range qrChan {
			if evt.Event == "code" {
				glog.Printf("scan below QRCode to add program as external device of your WhatsApp account:\n")
				qrterminal.GenerateHalfBlock(evt.Code, qrterminal.L, os.Stdout)
			} else {
				glog.Printf("login status %v\n", evt.Event)
			}
		}
		glog.Printf("successfully initiated new WhatsApp auth data to file %v and connected successfully\n", kWhatsAppAuthFilePath)
	} else {
		err := client.Connect()
		if err != nil {
			return nil, fmt.Errorf("error from client.Connect: %w", err)
		}
		glog.Printf("successfully connected to WhatsApp using auth data from file %v\n", kWhatsAppAuthFilePath)
	}
	return client, nil
}

package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"

	"go.mau.fi/whatsmeow"
	waProto "go.mau.fi/whatsmeow/binary/proto"
	"go.mau.fi/whatsmeow/types"
	"google.golang.org/api/gmail/v1"
	"google.golang.org/protobuf/proto"
)

// io.WriteCloser interface
type Sender struct {
	GmailClient    *gmail.Service
	WhatsAppClient *whatsmeow.Client

	from      string
	to        string
	targetJID types.JID
}

func NewSender(sendConfig SendConfig) (Sender, error) {
	var sender Sender
	var err error
	switch sendConfig.SendAction {
	case NoSend:
		return sender, nil
	case SendEmail:
		sender.GmailClient, err = NewGmailClient(sendConfig.EmailConfig)
		if err != nil {
			return sender, fmt.Errorf("error from NewGmailService: %w", err)
		}
		sender.from = sendConfig.EmailConfig.EmailFrom
		sender.to = sendConfig.EmailConfig.EmailTo
	case SendWhatsApp:
		sender.WhatsAppClient, err = NewWhatsAppClient(sendConfig.WhatsAppConfig)
		if err != nil {
			return sender, fmt.Errorf("error from NewWhatsAppClient: %w", err)
		}

		if len(sendConfig.WhatsAppConfig.GroupNameTo) > 0 {
			// Getting all the groups and contacts
			groups, err := sender.WhatsAppClient.GetJoinedGroups()
			if err != nil {
				return sender, fmt.Errorf("error from GetJoinedGroups: %w", err)
			}
			for _, group := range groups {
				if group.Name == sendConfig.WhatsAppConfig.GroupNameTo {
					sender.to = sendConfig.WhatsAppConfig.GroupNameTo
					sender.targetJID = group.JID
					break
				}
			}
		} else if len(sendConfig.WhatsAppConfig.UserNameTo) > 0 {
			users, err := sender.WhatsAppClient.Store.Contacts.GetAllContacts()
			if err != nil {
				return sender, fmt.Errorf("error from Store.Contacts.GetAllContacts: %w", err)
			}
			for jidType, contactInfo := range users {
				if contactInfo.FullName == sendConfig.WhatsAppConfig.UserNameTo {
					sender.to = sendConfig.WhatsAppConfig.UserNameTo
					sender.targetJID = jidType
					break
				}
			}
		} else {
			return sender, fmt.Errorf("at least group or user should be specified for WhatsApp message destination")
		}
	}
	return sender, nil
}

// satisfy the Writer interface
func (s Sender) Write(p []byte) (n int, err error) {
	nbBytesWritten := 0
	if s.GmailClient != nil {
		// New message for our gmail service to send
		var message gmail.Message

		// Compose the message
		messageBuf := bytes.NewBufferString(fmt.Sprintf(
			"From: %s\r\n"+
				"To: %s\r\n"+
				"Subject: [Too good to go] - Available bags!\r\n\r\n", s.from, s.to))

		_, err := messageBuf.Write(p)
		if err != nil {
			return nbBytesWritten, fmt.Errorf("error from messageBuf.Write: %w", err)
		}

		// Place messageStr into message.Raw in base64 encoded format
		message.Raw = base64.URLEncoding.EncodeToString(messageBuf.Bytes())

		// Send the message
		_, err = s.GmailClient.Users.Messages.Send("me", &message).Do()
		if err != nil {
			return nbBytesWritten, fmt.Errorf("error from gmailService.Users.Messages.Send: %v", err)
		}
		glog.Printf("email sent to %v\n", s.to)
		nbBytesWritten += messageBuf.Len()
	}
	if s.WhatsAppClient != nil {
		_, err := s.WhatsAppClient.SendMessage(context.Background(), s.targetJID, &waProto.Message{
			Conversation: proto.String(string(p)),
		})
		if err != nil {
			return nbBytesWritten, fmt.Errorf("error from s.WhatsAppClient.SendMessage: %w", err)
		}
		glog.Printf("whats app message sent to %v\n", s.to)
		nbBytesWritten += len(p)
	}

	return nbBytesWritten, nil
}

// satisfy the Closer interface
func (s Sender) Close() error {
	if s.WhatsAppClient != nil {
		s.WhatsAppClient.Disconnect()
	}
	return nil
}

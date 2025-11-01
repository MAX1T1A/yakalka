package workers

import (
	"context"
	"log"
	"slices"
	"strings"
	"sync/atomic"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types/events"
)

var msgCount atomic.Int64

func MsgCount() int64 {
	return msgCount.Load()
}

type WhatsAppMessage struct {
	ContactName string
	MessageText string
	IsGroup     bool
	ChatID      string
}

func HandleIncomingMessage(ctx context.Context, client *whatsmeow.Client, ch chan<- WhatsAppMessage, jids, triggerTexts []string, ev *events.Message) {
	select {
	case <-ctx.Done():
		return
	default:
	}

	if jids != nil {
		if !slices.Contains(jids, ev.Info.Chat.String()) {
			return
		}
	}

	messageText := ""
	if ev.Message.Conversation != nil {
		messageText = *ev.Message.Conversation
	} else if ev.Message.ExtendedTextMessage != nil &&
		ev.Message.ExtendedTextMessage.Text != nil {
		messageText = *ev.Message.ExtendedTextMessage.Text
	}

	if triggerTexts != nil {
		if !slices.Contains(triggerTexts, strings.ToLower(messageText)) {
			return
		}
	}

	user := ev.Info.Chat
	if ev.Info.IsGroup {
		user = ev.Info.Sender
	}

	contact, err := client.Store.Contacts.GetContact(ctx, user)
	if err != nil {
		log.Printf("ошибка получения информации о контакте: %v", err)
		return
	}

	contactName := contact.FullName
	if contactName == "" {
		contactName = contact.PushName
	}
	if contactName == "" {
		contactName = contact.BusinessName
	}
	if contactName == "" {
		contactName = user.String()
	}

	msg := WhatsAppMessage{
		ContactName: contactName,
		MessageText: messageText,
		IsGroup:     ev.Info.IsGroup,
		ChatID:      ev.Info.Chat.String(),
	}

	msgCount.Add(1)

	select {
	case ch <- msg:
	case <-ctx.Done():
	}
}

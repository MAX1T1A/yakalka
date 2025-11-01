package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
	"yakalka/config"
	"yakalka/workers"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types/events"
	waLog "go.mau.fi/whatsmeow/util/log"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	_ "github.com/mattn/go-sqlite3"
	"github.com/skip2/go-qrcode"
)

var logger = waLog.Stdout("wa", "INFO", true)

func main() {
	ctx, ctxCancel := context.WithCancel(context.Background())
	defer ctxCancel()

	cfgTgBot, cfgWhatsApp := config.LoadTgBotConfig(), config.LoadWhatsAppConfig()

	tgBot, err := tgbotapi.NewBotAPI(cfgTgBot.TelegramBotToken)
	if err != nil {
		log.Fatalf("Ошибка инициализации Telegram бота: %v", err)
	}

	chWhatsappMessages := make(chan workers.WhatsAppMessage, 100)

	consumerConfig := workers.ConsumerConfig{
		WorkerCount:  2,
		BatchSize:    3,
		BatchTimeout: 5 * time.Second,
	}

	go workers.Consumer(ctx, chWhatsappMessages, consumerConfig, tgBot, cfgTgBot.TelegramChatID)

	container, err := sqlstore.New(ctx, "sqlite3", "file:data/whatsmeow_session.db?_foreign_keys=on", logger)
	if err != nil {
		log.Fatalf("failed to create sql store: %v", err)
	}

	deviceStore, err := container.GetFirstDevice(ctx)
	if err != nil {
		log.Fatalf("failed to get device: %v", err)
	}

	client := whatsmeow.NewClient(deviceStore, logger)
	defer client.Disconnect()

	client.AddEventHandler(func(evt any) {
		switch v := evt.(type) {
		case *events.QR:
			code := v.Codes[0]
			filename := "data/qr.png"
			err := qrcode.WriteFile(code, qrcode.Medium, 256, filename)
			if err != nil {
				log.Printf("Ошибка генерации QR: %v", err)
				return
			}

		case *events.Connected:
			log.Println("Подключено к WhatsApp!")

		case *events.Disconnected:
			log.Println("Отключено от WhatsApp.")

		case *events.Message:
			go workers.HandleIncomingMessage(
				ctx,
				client,
				chWhatsappMessages,
				cfgWhatsApp.WhatsAppChatJIDs,
				cfgWhatsApp.TriggerTexts,
				v,
			)
		}
	})

	err = client.Connect()
	if err != nil {
		log.Fatalf("failed to connect: %v", err)
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	ctxCancel()

	close(chWhatsappMessages)
}

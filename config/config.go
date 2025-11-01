package config

import (
	"log"
	"os"
	"strconv"
	"strings"
)

type TgBotConfig struct {
	TelegramBotToken string
	TelegramChatID   int64
}

type WhatsAppConfig struct {
	WhatsAppChatJIDs []string
	TriggerTexts     []string
}

func LoadTgBotConfig() *TgBotConfig {
	cfg := &TgBotConfig{}

	cfg.TelegramBotToken = os.Getenv("TELEGRAM_BOT_TOKEN")
	if cfg.TelegramBotToken == "" {
		log.Fatal("TELEGRAM_BOT_TOKEN не установлен в переменных окружения")
	}

	chatIDStr := os.Getenv("TELEGRAM_CHAT_ID")
	if chatIDStr == "" {
		log.Fatal("TELEGRAM_CHAT_ID не установлен в переменных окружения")
	}

	chatID, err := strconv.ParseInt(chatIDStr, 10, 64)
	if err != nil {
		log.Fatalf("Неверный формат TELEGRAM_CHAT_ID: %v", err)
	}
	cfg.TelegramChatID = chatID

	return cfg
}

func LoadWhatsAppConfig() *WhatsAppConfig {
	cfg := &WhatsAppConfig{}

	whatsAppChatJIDs := os.Getenv("WHATSAPP_CHAT_JIDS")
	if whatsAppChatJIDs == "" {
		cfg.WhatsAppChatJIDs = nil
	} else {
		cfg.WhatsAppChatJIDs = strings.Split(whatsAppChatJIDs, ",")
	}

	triggerTexts := os.Getenv("TRIGGER_TEXTS")
	if triggerTexts == "" {
		cfg.TriggerTexts = nil
	} else {
		cfg.TriggerTexts = strings.Split(triggerTexts, ",")
	}

	return cfg
}

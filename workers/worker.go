package workers

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type ConsumerConfig struct {
	WorkerCount  int
	BatchSize    int
	BatchTimeout time.Duration
}

func Consumer(ctx context.Context, ch <-chan WhatsAppMessage, config ConsumerConfig, tgBot *tgbotapi.BotAPI, tgChatID int64) {
	wg := &sync.WaitGroup{}
	defer wg.Wait()

	for i := range config.WorkerCount {
		wg.Add(1)
		go consumerWorker(ctx, wg, ch, i, config, tgBot, tgChatID)
	}
}

func consumerWorker(ctx context.Context, wg *sync.WaitGroup, ch <-chan WhatsAppMessage, workerID int, config ConsumerConfig, tgBot *tgbotapi.BotAPI, tgChatID int64) {
	defer wg.Done()

	batch := make([]WhatsAppMessage, 0, config.BatchSize)
	ticker := time.NewTicker(config.BatchTimeout)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			if len(batch) > 0 {
				sendBatchToTelegram(workerID, batch, tgBot, tgChatID)
			}
			return

		case <-ticker.C:
			if len(batch) > 0 {
				sendBatchToTelegram(workerID, batch, tgBot, tgChatID)
				batch = batch[:0]
				ticker.Reset(config.BatchTimeout)
			}

		case msg, ok := <-ch:
			if !ok {
				if len(batch) > 0 {
					sendBatchToTelegram(workerID, batch, tgBot, tgChatID)
				}
				return
			}

			batch = append(batch, msg)

			if len(batch) >= config.BatchSize {
				sendBatchToTelegram(workerID, batch, tgBot, tgChatID)
				batch = batch[:0]
				ticker.Reset(config.BatchTimeout)
			}
		}
	}
}

func sendBatchToTelegram(workerID int, batch []WhatsAppMessage, tgBot *tgbotapi.BotAPI, tgChatID int64) {
	if len(batch) == 0 {
		return
	}

	var messageBuilder strings.Builder
	messageBuilder.WriteString(fmt.Sprintf(
		"📨 <b>Сообщения WhatsApp</b> (%d)\n<i>%s</i>\n\n",
		len(batch),
		time.Now().Format("15:04:05"),
	))

	for i, msg := range batch {
		text := msg.MessageText
		if len(text) > 120 {
			text = text[:120] + "..."
		}

		prefix := "👤"
		if msg.IsGroup {
			prefix = "👥"
		}

		messageBuilder.WriteString(fmt.Sprintf(
			"%d. %s <b>%s</b>\n    💬%s\n\n",
			i+1, prefix, msg.ContactName, text,
		))
	}

	message := messageBuilder.String()
	if len(message) > 4000 {
		sendLongBatchToTelegram(workerID, batch, tgBot, tgChatID)
		return
	}

	tgMsg := tgbotapi.NewMessage(tgChatID, message)
	tgMsg.ParseMode = "HTML"

	_, err := tgBot.Send(tgMsg)
	if err != nil {
		log.Printf("[Consumer-%d] Ошибка отправки батча в Telegram: %v", workerID, err)
	} else {
		log.Printf("[Consumer-%d] Батч из %d сообщений отправлен в Telegram", workerID, len(batch))
	}
}

func sendLongBatchToTelegram(workerID int, batch []WhatsAppMessage, tgBot *tgbotapi.BotAPI, tgChatID int64) {
	log.Printf("[Consumer-%d] Батч слишком длинный, отправляем частями\n", workerID)
	chunkSize := 3
	for i := 0; i < len(batch); i += chunkSize {
		end := min(i+chunkSize, len(batch))
		chunk := batch[i:end]
		sendBatchToTelegram(workerID, chunk, tgBot, tgChatID)

		if end < len(batch) {
			time.Sleep(500 * time.Millisecond)
		}
	}
}

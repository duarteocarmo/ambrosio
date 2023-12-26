package main

import (
	"log"
	"os"

	"duarteocarmo/ambrosio/modes"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func main() {

	bot, err := tgbotapi.NewBotAPI(os.Getenv("TELEGRAM_APITOKEN"))

	if err != nil {
		log.Panic(err)
	}

	bot.Debug = os.Getenv("DEBUG") == "true"

	log.Printf("Authorized on account %s", bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := bot.GetUpdatesChan(u)

	for update := range updates {
		if update.Message == nil {
			continue
		}

		if !update.Message.IsCommand() {
			continue
		}

		chatID := update.Message.Chat.ID
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, "")

		switch update.Message.Command() {

		case "photo":
			msg.Text = "Entering photo mode..."
			bot.Send(msg)
			modes.PhotoMode(updates, bot, chatID)
			continue
		case "assistant":
			msg.Text = "Placeholder for the assistant mode"

		default:
			msg.Text = "I don't know that command"
		}

		if _, err := bot.Send(msg); err != nil {
			log.Panic(err)
		}
	}
}

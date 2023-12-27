package main

import (
	"fmt"
	"log"
	"os"
	"strings"

	"duarteocarmo/ambrosio/modes"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

const (
	PhotoMode     = "photo"
	AssistantMode = "assistant"
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
	availableModes := []string{PhotoMode, AssistantMode}
	helpMsg := fmt.Sprintf("I don't know that command. Available commands are: \n /%s", strings.Join(availableModes, "\n"))

	for update := range updates {

		if update.Message == nil {
			continue
		}

		chatID := update.Message.Chat.ID
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, "")

		if update.SentFrom().UserName != os.Getenv("TELEGRAM_USERNAME") {
			msg.Text = "You are not authorized to use this bot"
			bot.Send(msg)
			log.Printf("Unauthorized user %s", update.SentFrom().UserName)
			continue
		}

		if !update.Message.IsCommand() {
			msg.Text = helpMsg
			bot.Send(msg)
			continue
		}

		switch update.Message.Command() {

		case PhotoMode:
			msg.Text = "Entering photo mode..."
			bot.Send(msg)
			modes.PhotoMode(updates, bot, chatID)
			continue
		case AssistantMode:
			msg.Text = "Placeholder for the assistant mode"

		default:
			msg.Text = helpMsg

		}

		if _, err := bot.Send(msg); err != nil {
			log.Panic(err)
		}
	}
}

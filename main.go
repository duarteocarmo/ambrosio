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
	Timeout       = 60
)

func createBot() (*tgbotapi.BotAPI, error) {
	m := os.Getenv("MODE")
	t := ""
	d := false

	switch m {
	case "DEV":
		t = os.Getenv("TELEGRAM_APITOKEN_DEV")
		d = true
		log.Println("Running in DEV mode")
	case "PROD":
		t = os.Getenv("TELEGRAM_APITOKEN_PROD")
		d = false
		log.Println("Running in PROD mode")
	default:
		log.Println("No mode specified. Exiting...")
		return nil, fmt.Errorf("no mode specified")
	}

	bot, err := tgbotapi.NewBotAPI(t)

	if err != nil {
		return nil, fmt.Errorf("error creating bot: %v", err)
	}

	bot.Debug = d

	log.Printf("Authorized on account %s", bot.Self.UserName)
	log.Printf("Bot is running")

	return bot, nil

}

func main() {
	bot, err := createBot()
	if err != nil {
		log.Panicf("Error creating bot: %v", err)
	}

	u := tgbotapi.NewUpdate(0)
	u.Timeout = Timeout
	updates := bot.GetUpdatesChan(u)
	authorizedUser := os.Getenv("TELEGRAM_USERNAME")

	availableModes := []string{PhotoMode, AssistantMode}
	helpMsg := "I don't know that command. Available commands are: \n/" + strings.Join(availableModes, "\n/")

	for update := range updates {

		if update.Message == nil {
			continue
		}

		msg := tgbotapi.NewMessage(update.Message.Chat.ID, "")

		if update.SentFrom().UserName != authorizedUser {
			msg.Text = "Sorry, you are not authorized to use this bot"
			bot.Send(msg)
			log.Printf("Detected unauthorized user %s", update.SentFrom().UserName)
			continue
		}

		if !update.Message.IsCommand() {
			msg.Text = helpMsg
			bot.Send(msg)
			continue
		}

		switch update.Message.Command() {
		case PhotoMode:
			err := modes.PhotoMode(update, updates, bot)
			if err != nil {
				log.Printf("Error in photo mode: %v", err)
				msg.Text = fmt.Sprintf("Error in photo mode: %v", err)
				bot.Send(msg)
			}
			continue
		case AssistantMode:
			msg.Text = "Placeholder for the assistant mode"
			// TODO: Implement assistant mode
		default:
			msg.Text = helpMsg

		}

		if _, err := bot.Send(msg); err != nil {
			log.Panicf("Error sending message: %v", err)
		}
	}
}

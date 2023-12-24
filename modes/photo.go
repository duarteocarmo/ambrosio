package modes

import (
	"log"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

const (
	PhotoModeCreate = "create"
	PhotoModeEdit   = "edit"
	PhotoModeDelete = "delete"
	PhotoModeExit   = "exit"
)

var optionKeyboard = tgbotapi.NewInlineKeyboardMarkup(
	tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData(PhotoModeEdit, PhotoModeEdit),
		tgbotapi.NewInlineKeyboardButtonData(PhotoModeDelete, PhotoModeDelete),
	),
	tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData(PhotoModeCreate, PhotoModeCreate),
		tgbotapi.NewInlineKeyboardButtonData(PhotoModeExit, PhotoModeExit),
	),
)

func PhotoMode(updates tgbotapi.UpdatesChannel, bot *tgbotapi.BotAPI, chatID int64) {

	msg := tgbotapi.NewMessage(chatID, "Photo mode")
	msg.ReplyMarkup = optionKeyboard

	if _, err := bot.Send(msg); err != nil {
		panic(err)
	}

	for update := range updates {

		if update.CallbackQuery == nil {
			log.Printf("CallbackQuery")
			msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Select an option")
			msg.ReplyMarkup = optionKeyboard

			if _, err := bot.Send(msg); err != nil {
				panic(err)
			}
			continue
		}

		msg := tgbotapi.NewMessage(update.CallbackQuery.Message.Chat.ID, "")
		action := update.CallbackQuery.Data

		switch action {
		case PhotoModeCreate:
			msg.Text = "Creating photo..."
		case PhotoModeEdit:
			msg.Text = "Editing photo..."
		case PhotoModeDelete:
			msg.Text = "Deleting photo..."
		case PhotoModeExit:
			msg.Text = "Exiting photo mode..."
			bot.Send(msg)
			return
		default:
			msg.Text = "Unknown command"
		}
		if _, err := bot.Send(msg); err != nil {
			panic(err)
		}
	}
}

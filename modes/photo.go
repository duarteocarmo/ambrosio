package modes

import (
	"crypto/sha1"
	"duarteocarmo/ambrosio/storage"
	"fmt"
	"strings"
	"time"

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
			createPhotoFlow(updates, bot, chatID)
			return
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

func createPhotoFlow(updates tgbotapi.UpdatesChannel, bot *tgbotapi.BotAPI, chatID int64) {

	p := storage.Photo{}

	// receive photo
	sendMessage(tgbotapi.Update{Message: &tgbotapi.Message{Chat: &tgbotapi.Chat{ID: chatID}}}, bot, "Waiting to receive photo...")
	for update := range updates {
		for update.Message.Photo == nil {
			sendMessage(update, bot, "That's not a photo.")
		}
		photoURL := getPhotoDownloadUrl(update, bot)
		p.Url = photoURL
		sendMessage(tgbotapi.Update{Message: &tgbotapi.Message{Chat: &tgbotapi.Chat{ID: chatID}}}, bot, "Photo received successfully.")
		break
	}

	// receive caption
	sendMessage(tgbotapi.Update{Message: &tgbotapi.Message{Chat: &tgbotapi.Chat{ID: chatID}}}, bot, "Waiting to receive caption...")
	for update := range updates {
		if strings.ToLower(update.Message.Text) == "skip" {
			sendMessage(update, bot, "Caption will be empty.")
		} else if update.Message.Text != "" {
			caption := update.Message.Text
			p.Caption = &caption
			sendMessage(update, bot, "Caption received successfully: "+caption)
		} else {
			sendMessage(update, bot, "That's not a caption.")
		}

		break
	}

	// receive location
	sendMessage(tgbotapi.Update{Message: &tgbotapi.Message{Chat: &tgbotapi.Chat{ID: chatID}}}, bot, "Waiting to receive location...")
	for update := range updates {
		if strings.ToLower(update.Message.Text) == "skip" {
			sendMessage(update, bot, "Location will be empty.")
		} else if update.Message.Venue.Title != "" {
			location := update.Message.Venue
			p.Location = &location.Title
			sendMessage(update, bot, "Location received successfully: "+location.Title)
		} else {
			sendMessage(update, bot, "That's not a location.")
		}

		break
	}

	err := p.Create()
	if err != nil {
		panic(err)
	}
	sendMessage(tgbotapi.Update{Message: &tgbotapi.Message{Chat: &tgbotapi.Chat{ID: chatID}}}, bot, "Photo created successfully.")

	return

}

func getPhotoDownloadUrl(update tgbotapi.Update, bot *tgbotapi.BotAPI) string {
	p := update.Message.Photo

	fileID := update.Message.Photo[len(p)-1].FileID
	file, error := bot.GetFile(tgbotapi.FileConfig{FileID: fileID})
	if error != nil {
		panic(error)
	}

	filePath := file.FilePath
	downloadURL := "https://api.telegram.org/file/bot" + bot.Token + "/" + filePath

	return downloadURL
}

func sendMessage(update tgbotapi.Update, bot *tgbotapi.BotAPI, text string) {
	msg := tgbotapi.NewMessage(update.Message.Chat.ID, text)
	if _, err := bot.Send(msg); err != nil {
		panic(err)
	}
}

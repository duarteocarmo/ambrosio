package modes

import (
	"duarteocarmo/ambrosio/storage"
	"strings"

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
		tgbotapi.NewInlineKeyboardButtonData(PhotoModeCreate, PhotoModeCreate),
		tgbotapi.NewInlineKeyboardButtonData(PhotoModeDelete, PhotoModeDelete),
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
			deletePhotoFlow(updates, bot, chatID)
			return
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

func deletePhotoFlow(updates tgbotapi.UpdatesChannel, bot *tgbotapi.BotAPI, chatID int64) {

	sendMessage(tgbotapi.Update{Message: &tgbotapi.Message{Chat: &tgbotapi.Chat{ID: chatID}}}, bot, "Please send the photo ID to delete")
	for update := range updates {
		switch {
		case strings.ToLower(update.Message.Text) == "skip":
			sendMessage(update, bot, "Aborting")
			return
		case update.Message.Text != "":
			id := update.Message.Text
			msg, err := storage.DeletePhoto(id)
			if err != nil {
				sendMessage(update, bot, "Error deleting photo: "+err.Error())
				continue
			}
			sendMessage(update, bot, msg)
			break

		default:
			sendMessage(update, bot, "That's not a valid ID.")
			continue
		}
		break
	}

}

func createPhotoFlow(updates tgbotapi.UpdatesChannel, bot *tgbotapi.BotAPI, chatID int64) {

	p := storage.Photo{}

	// receive photo
	sendMessage(tgbotapi.Update{Message: &tgbotapi.Message{Chat: &tgbotapi.Chat{ID: chatID}}}, bot, "Please send a photo")
	for update := range updates {
		switch {
		case update.Message.Photo == nil:
			sendMessage(update, bot, "That's not a photo.")
			continue
		case update.Message.Text == "skip":
			sendMessage(update, bot, "Aborting")
			return
		default:
			photoURL := getPhotoDownloadUrl(update, bot)
			p.Url = photoURL
			sendMessage(tgbotapi.Update{Message: &tgbotapi.Message{Chat: &tgbotapi.Chat{ID: chatID}}}, bot, "Photo received successfully.")
			break
		}
		break
	}

	// receive caption
	sendMessage(tgbotapi.Update{Message: &tgbotapi.Message{Chat: &tgbotapi.Chat{ID: chatID}}}, bot, "Please send a caption")
	for update := range updates {
		switch {
		case strings.ToLower(update.Message.Text) == "skip":
			sendMessage(update, bot, "Caption will be empty.")
			break
		case update.Message.Text != "":
			caption := update.Message.Text
			p.Caption = &caption
			sendMessage(update, bot, "Caption received successfully: "+caption)
		default:
			sendMessage(update, bot, "That's not a caption.")
			continue
		}
		break
	}

	// receive location
	sendMessage(tgbotapi.Update{Message: &tgbotapi.Message{Chat: &tgbotapi.Chat{ID: chatID}}}, bot, "Waiting to receive location...")
	for update := range updates {
		switch {
		case update.Message.Venue != nil && update.Message.Venue.Title != "":
			p.Location = &update.Message.Venue.Title
		case strings.ToLower(update.Message.Text) == "skip":
			sendMessage(update, bot, "Location will be empty.")
			break
		case update.Message.Text != "":
			p.Location = &update.Message.Text
		default:
			sendMessage(update, bot, "That's not a location.")
			continue
		}
		if p.Location != nil {
			sendMessage(update, bot, "Location received successfully: "+*p.Location)
		}
		break
	}

	msg, err := p.Create()
	if err != nil {
		panic(err)
	}
	sendMessage(tgbotapi.Update{Message: &tgbotapi.Message{Chat: &tgbotapi.Chat{ID: chatID}}}, bot, msg)

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

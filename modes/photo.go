package modes

import (
	"io"
	"log"
	"net/http"
	"os"

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

	msg := tgbotapi.NewMessage(chatID, "Please send me a photo")
	if _, err := bot.Send(msg); err != nil {
		panic(err)
	}

	for update := range updates {

		if update.Message.Command() == PhotoModeExit {
			return
		} else if update.Message.Photo != nil {
			log.Printf("Photo: %+v\n", update.Message.Photo)
			p := update.Message.Photo

			fileID := update.Message.Photo[len(p)-1].FileID
			file, error := bot.GetFile(tgbotapi.FileConfig{FileID: fileID})
			if error != nil {
				panic(error)
			}

			filePath := file.FilePath
			downloadURL := "https://api.telegram.org/file/bot" + bot.Token + "/" + filePath
			savePhoto(downloadURL)

			sendMessage(update, bot, "File downloaded successfully.")
			return

		} else {
			sendMessage(update, bot, "That's not a photo.")
			log.Printf("Not a photo: %+v\n", update.Message)
			continue

		}

	}

}

func promptPhotoDescription(updates tgbotapi.UpdatesChannel, bot *tgbotapi.BotAPI, chatID int64) (repsponse string, ok bool) {

	msg := tgbotapi.NewMessage(chatID, "Please send me a description")
	if _, err := bot.Send(msg); err != nil {
		panic(err)
	}

	for update := range updates {

		if update.Message.Command() == PhotoModeExit {
			return "", false
		} else if update.Message.Text != "" {
			log.Printf("Description: %+v\n", update.Message.Text)
			sendMessage(update, bot, "Description saved successfully.")
			return update.Message.Text, true

		} else {
			sendMessage(update, bot, "That's not a description.")
			log.Printf("Not a description: %+v\n", update.Message)
			continue

		}

	}

}

func sendMessage(update tgbotapi.Update, bot *tgbotapi.BotAPI, text string) {
	msg := tgbotapi.NewMessage(update.Message.Chat.ID, text)
	if _, err := bot.Send(msg); err != nil {
		panic(err)
	}
}

func savePhoto(url string) {
	resp, err := http.Get(url)
	if err != nil {
		log.Fatalf("Failed to download file: %v\n", err)
	}
	defer resp.Body.Close()

	out, err := os.Create("photo.png")
	if err != nil {
		log.Fatalf("Failed to create file on disk: %v\n", err)
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		log.Fatalf("Failed to write file to disk: %v\n", err)
	}

}

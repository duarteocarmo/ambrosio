package modes

import (
	"duarteocarmo/ambrosio/storage"
	"fmt"
	"log"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

const (
	PhotoModeCreate = "create"
	PhotoModeDelete = "delete"
	PhotoModeExit   = "exit"
)

func PhotoMode(currentUpdate tgbotapi.Update, updates tgbotapi.UpdatesChannel, bot *tgbotapi.BotAPI) error {

	chatID := currentUpdate.Message.Chat.ID

	if currentUpdate.Message == nil || currentUpdate.Message.Text == "" {
		return fmt.Errorf("the update does not contain a message or text")
	}

	textParts := strings.Split(currentUpdate.Message.Text, " ")
	if len(textParts) <= 1 {
		return fmt.Errorf("No action specified in the message text")
	}

	selectedAction := textParts[1]
	log.Printf("Selected action: %s", selectedAction)

	switch selectedAction {
	case PhotoModeCreate:
		err := createPhotoFlow(updates, bot, chatID)
		if err != nil {
			return fmt.Errorf("error creating photo: %v", err)
		}
		return nil

	case PhotoModeDelete:
		deletePhotoFlow(updates, bot, chatID)
		return nil

	default:
		msg := tgbotapi.NewMessage(chatID, "Unknown command, please try again")
		bot.Send(msg)
		return nil
	}

}

func deletePhotoFlow(updates tgbotapi.UpdatesChannel, bot *tgbotapi.BotAPI, chatID int64) error {

	sendMessage(tgbotapi.Update{Message: &tgbotapi.Message{Chat: &tgbotapi.Chat{ID: chatID}}}, bot, "Please send the photo ID to delete")

	for update := range updates {
		switch {

		case strings.ToLower(update.Message.Text) == PhotoModeExit:
			sendMessage(update, bot, "Aborting")
			return nil

		case update.Message.Text != "":
			id := update.Message.Text
			msg, err := storage.DeletePhoto(id)
			if err != nil {
				return fmt.Errorf("error deleting photo: %v", err)
			}
			sendMessage(update, bot, msg)
			return nil

		default:
			sendMessage(update, bot, "That's not a valid ID.")
			return fmt.Errorf("invalid ID")
		}
	}

	return nil

}

func createPhotoFlow(updates tgbotapi.UpdatesChannel, bot *tgbotapi.BotAPI, chatID int64) error {

	p := storage.Photo{}

	// receive photo
	sendMessage(tgbotapi.Update{Message: &tgbotapi.Message{Chat: &tgbotapi.Chat{ID: chatID}}}, bot, "Please send a photo")
	for update := range updates {
		switch {
		case strings.ToLower(update.Message.Text) == PhotoModeExit:
			sendMessage(update, bot, "Aborting")
			return nil
		case update.Message.Photo == nil:
			sendMessage(update, bot, "That's not a photo.")
			continue
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
		case strings.ToLower(update.Message.Text) == PhotoModeExit:
			sendMessage(update, bot, "Aborting")
			return nil
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
		case strings.ToLower(update.Message.Text) == PhotoModeExit:
			sendMessage(update, bot, "Aborting")
			return nil
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

	return nil

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

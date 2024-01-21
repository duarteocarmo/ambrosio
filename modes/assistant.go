package modes

import (
	"bytes"
	"duarteocarmo/ambrosio/model"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"image"
	"image/png"
	"io"
	"log"
	"net/http"
	"os"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

const (
	TogetherEndpoint = "https://api.together.xyz/inference"
	ChatEndpoint     = "https://api.together.xyz/v1/chat/completions"
	ModelID          = "mistralai/Mixtral-8x7B-Instruct-v0.1"
	PhotoGenModelID  = "stabilityai/stable-diffusion-xl-base-1.0"
	// PhotoGenModelID  = "stabilityai/stable-diffusion-2-1"
	ChatMode         = "chat"
	PhotoGenMode     = "photo"
	ExitCommand      = "exit"
	ResetCommand     = "reset"
)

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ApiResponse struct {
	Choices []struct {
		Message `json:"message"`
	} `json:"choices"`
}

type ApiPhotoGenResponse struct {
	Output struct {
		Choices []struct {
			Image string `json:"image_base64"`
		} `json:"choices"`
	} `json:"output"`
}

type ApiRequest struct {
	Model             string    `json:"model"`
	MaxTokens         int       `json:"max_tokens"`
	Stop              []string  `json:"stop"`
	Temperature       float32   `json:"temperature"`
	TopP              float32   `json:"top_p"`
	TopK              int       `json:"top_k"`
	RepetitionPenalty float32   `json:"repetition_penalty"`
	N                 int       `json:"n"`
	Messages          []Message `json:"messages"`
}

func AssistantMode(currentUpdate tgbotapi.Update, updates tgbotapi.UpdatesChannel, bot *tgbotapi.BotAPI) error {

	chatID := currentUpdate.Message.Chat.ID
	supportedModes := []string{ChatMode, PhotoGenMode}
	noActionError := fmt.Errorf("No action specified, provide one of the following: %s", strings.Join(supportedModes, ", "))

	if currentUpdate.Message == nil || currentUpdate.Message.Text == "" {
		return noActionError
	}

	textParts := strings.Split(currentUpdate.Message.Text, " ")
	if len(textParts) <= 1 {
		return noActionError
	}

	selectedAction := textParts[1]
	log.Printf("Selected action: %s", selectedAction)

	switch selectedAction {
	case ChatMode:
		err := chatFlow(updates, bot, chatID)
		if err != nil {
			return fmt.Errorf("error creating photo: %v", err)
		}
		return nil

	case PhotoGenMode:
		err := photogenFlow(updates, bot, chatID)
		if err != nil {
			return fmt.Errorf("error creating photo: %v", err)
		}
		return nil

	default:
		msg := tgbotapi.NewMessage(chatID, "Unknown command, please try again")
		bot.Send(msg)
		return nil
	}

}

func chatFlow(updates tgbotapi.UpdatesChannel, bot *tgbotapi.BotAPI, chatID int64) error {

	bot.Send(tgbotapi.NewMessage(chatID, "Assistant mode activated."))

	systemPrompt, err := model.LoadPromptFromFile("system")
	if err != nil {
		return err
	}

	messages := []Message{}
	messages = append(messages, Message{Role: "system", Content: systemPrompt})

	for update := range updates {

		messageText := update.Message.Text

		if messageText == "" || update.Message == nil {
			bot.Send(tgbotapi.NewMessage(chatID, "Please send a text message."))
			continue
		}

		if strings.ToLower(messageText) == ExitCommand {
			bot.Send(tgbotapi.NewMessage(chatID, "Assistant mode deactivated."))
			return nil
		}

		if strings.ToLower(messageText) == ResetCommand {
			messages = []Message{}
			messages = append(messages, Message{Role: "system", Content: systemPrompt})
			bot.Send(tgbotapi.NewMessage(chatID, "* Prompt reset *"))
			continue
		}

		messages = append(messages, Message{Role: "user", Content: messageText})

		bot.Send(tgbotapi.NewChatAction(chatID, "typing"))

		assistantMessage, err := makeChatRequest(
			messages,
		)

		if err != nil {
			bot.Send(tgbotapi.NewMessage(chatID, fmt.Sprintf("Error: %v", err)))
		} else {
			msg := tgbotapi.NewMessage(chatID, assistantMessage.Content)
			msg.ParseMode = "Markdown"
			bot.Send(msg)
			messages = append(messages, assistantMessage)

		}

	}

	return nil
}

func makeChatRequest(
	messages []Message,
) (Message, error) {

	apiRequest := ApiRequest{
		Model:             ModelID,
		MaxTokens:         512,
		Stop:              []string{"</s>", "[/INST]"},
		Temperature:       0.0,
		TopP:              0.7,
		TopK:              50,
		RepetitionPenalty: 1,
		N:                 1,
		Messages:          messages,
	}

	apiKey := os.Getenv("TOGETHER_API_KEY")

	if apiKey == "" {
		return Message{}, fmt.Errorf("TOGETHER_API_KEY environment variable not set")
	}

	body, err := sendPostRequest(apiRequest, apiKey, ChatEndpoint)
	if err != nil {
		return Message{}, err
	}

	var apiResponse ApiResponse
	err = json.Unmarshal(body, &apiResponse)
	if err != nil {
		return Message{}, err
	}

	if len(apiResponse.Choices) > 0 {
		return apiResponse.Choices[0].Message, nil
	}

	return Message{}, fmt.Errorf("no choices found in the response")

}

func photogenFlow(updates tgbotapi.UpdatesChannel, bot *tgbotapi.BotAPI, chatID int64) error {

	bot.Send(tgbotapi.NewMessage(chatID, "Photo generation mode activated. Go ahead and send your prompt."))

	for update := range updates {
		switch {

		case strings.ToLower(update.Message.Text) == ExitCommand:
			sendMessage(update, bot, "Aborting")
			return nil

		case update.Message.Text != "":
			genText := update.Message.Text
			bot.Send(tgbotapi.NewMessage(chatID, "Generating photo for text: "+genText))

			bot.Send(tgbotapi.NewChatAction(chatID, "typing"))
			imageBytes, err := makePhotoGenRequest(genText)
			if err != nil {
				sendMessage(update, bot, fmt.Sprintf("Error: %v", err))
				return err
			}

			mediaGroup := []interface{}{}
			for index, img := range imageBytes {
				mediaGroup = append(mediaGroup, tgbotapi.NewInputMediaPhoto(tgbotapi.FileBytes{Name: "image" + fmt.Sprintf("%d", index) + ".png", Bytes: img}))
			}

			bot.SendMediaGroup(tgbotapi.NewMediaGroup(chatID, mediaGroup))

			return nil

		default:
			sendMessage(update, bot, "That's not a text generation message,")
			return fmt.Errorf("Invalid message type")
		}
	}

	return nil
}

func makePhotoGenRequest(prompt string) ([][]byte, error) {

	negativePrompt := ""
	width := 1024
	height := 1024
	steps := 40
	n := 4
	seed := 9394
	apiKey := os.Getenv("TOGETHER_API_KEY")

	defaultURL := TogetherEndpoint

	payload := map[string]interface{}{
		"model":               PhotoGenModelID,
		"prompt":              prompt,
		"negative_prompt":     negativePrompt,
		"width":               width,
		"height":              height,
		"num_inference_steps": steps,
		"n":                   n,
		"seed":                seed,
		"steps":               steps,
	}

	body, err := sendPostRequest(payload, apiKey, defaultURL)
	if err != nil {
		return nil, err
	}

	var apiResponse ApiPhotoGenResponse
	err = json.Unmarshal(body, &apiResponse)
	if err != nil {
		return nil, err
	}

	// an array of arrays of bytes
	var images [][]byte

	if len(apiResponse.Output.Choices) != n {
		return nil, fmt.Errorf("Expected %d images, got %d", n, len(apiResponse.Output.Choices))
	}

	for _, choice := range apiResponse.Output.Choices {
		imgBytes, err := base64ToBytes(choice.Image)
		if err != nil {
			return nil, err
		}
		images = append(images, imgBytes)
	}

	return images, nil

}

func sendPostRequest(payload interface{}, apiKey, defaultURL string) ([]byte, error) {
	bytesPayload, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", defaultURL, bytes.NewReader(bytesPayload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return body, nil

}

func base64ToBytes(base64Str string) ([]byte, error) {
	data, err := base64.StdEncoding.DecodeString(base64Str)
	if err != nil {
		return nil, err
	}

	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}

	buf := new(bytes.Buffer)
	err = png.Encode(buf, img)
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

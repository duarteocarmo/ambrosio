package modes

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func AssistantMode(currentUpdate tgbotapi.Update, updates tgbotapi.UpdatesChannel, bot *tgbotapi.BotAPI) error {

	chatID := currentUpdate.Message.Chat.ID
	bot.Send(tgbotapi.NewMessage(chatID, "Assistant mode activated."))

	for update := range updates {

		messageText := update.Message.Text

		if messageText == "" || update.Message == nil {
			bot.Send(tgbotapi.NewMessage(chatID, "Please send a text message."))
			continue
		}

		if strings.ToLower(messageText) == "exit" {
			bot.Send(tgbotapi.NewMessage(chatID, "Assistant mode deactivated."))
			return nil
		}

		// prompt := fmt.Sprintf("GPT4 User: %s<|end_of_turn|>GPT4 Assistant:", messageText)
		prompt := fmt.Sprintf("Instruct: %s\nOutput:", messageText)

		response, err := makeRequest(
			prompt,
			250,
			0.0,
			false,
			[]string{},
			false,
		)

		if err != nil {
			bot.Send(tgbotapi.NewMessage(chatID, fmt.Sprintf("Error: %v", err)))
		} else {
			bot.Send(tgbotapi.NewMessage(chatID, response))
		}

	}

	return nil
}

func makeRequest(
	prompt string,
	nPredict int,
	temperature float64,
	stream bool,
	stop []string,
	ignoreEos bool,
) (string, error) {

	endpoint := os.Getenv("COMPLETION_ENDPOINT")
	if endpoint == "" {
		return "", errors.New("COMPLETION_ENDPOINT not set")
	}

	defaultURL := fmt.Sprintf("%s/completion", endpoint)
	payload := map[string]interface{}{
		"prompt":      prompt,
		"n_predict":   nPredict,
		"temperature": temperature,
		"stream":      stream,
		"stop":        stop,
		"ignore_eos":  ignoreEos,
	}
	bytesPayload, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	log.Printf("Request payload: %s", payload)

	req, err := http.NewRequest("POST", defaultURL, bytes.NewReader(bytesPayload))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("request failed: %s", resp.Status)
	}

	var respPayload map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&respPayload); err != nil {
		return "", err
	}

	log.Printf("Response payload: %v", respPayload)

	content, ok := respPayload["content"].(string)
	if !ok {
		return "", errors.New("invalid response format")
	}

	return content, nil
}

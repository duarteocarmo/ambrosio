package modes

import (
	"bytes"
	"duarteocarmo/ambrosio/model"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

const (
	TogetherEndpoint = "https://api.together.xyz/inference"
	ModelID          = "mistralai/Mixtral-8x7B-Instruct-v0.1"
)

type APIResponse struct {
	Output struct {
		Choices []struct {
			Text string `json:"text"`
		} `json:"choices"`
	} `json:"output"`
}

func AssistantMode(currentUpdate tgbotapi.Update, updates tgbotapi.UpdatesChannel, bot *tgbotapi.BotAPI) error {

	chatID := currentUpdate.Message.Chat.ID
	bot.Send(tgbotapi.NewMessage(chatID, "Assistant mode activated."))

	systemPrompt, err := model.LoadPromptFromFile("system")
	if err != nil {
		return err
	}

	promptStart := "<s> [INST] %s [/INST]"
	prompt := fmt.Sprintf(promptStart, systemPrompt)

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

		if strings.ToLower(messageText) == "reset" {
			prompt = fmt.Sprintf(promptStart, systemPrompt)
			bot.Send(tgbotapi.NewMessage(chatID, "* Prompt reset *"))
			continue
		}

		prompt += fmt.Sprintf("%s [/INST]", messageText)

		response, err := makeRequest(
			prompt,
			ModelID,
			false,
		)

		if err != nil {
			bot.Send(tgbotapi.NewMessage(chatID, fmt.Sprintf("Error: %v", err)))
		} else {
			bot.Send(tgbotapi.NewMessage(chatID, response))
			prompt += fmt.Sprintf(" %s</s>", response)

		}

	}

	return nil
}

func makeRequest(
	prompt string,
	modelID string,
	streamTokens bool,
) (string, error) {

	defaultURL := TogetherEndpoint
	maxTokens := 512
	temperature := 0.0
	topP := 0.7
	topK := 50
	repetitionPenalty := 1.0
	apiKey := os.Getenv("TOGETHER_API_KEY")
	stop := []string{"</s>", "[INST]"}

	// negativePrompt := ""
	// requestType := "language-model-inference"

	if apiKey == "" {
		return "", fmt.Errorf("TOGETHER_API_KEY environment variable not set")
	}

	payload := map[string]interface{}{
		"model":              modelID,
		"prompt":             prompt,
		"temperature":        temperature,
		"top_p":              topP,
		"top_k":              topK,
		"max_tokens":         maxTokens,
		"repetition_penalty": repetitionPenalty,
		"stop":               stop,

		// "stream_tokens":      streamTokens,
		// "negative_prompt":    negativePrompt,
		// "sessionKey":         sessionKey,
	}
	bytesPayload, err := json.Marshal(payload)

	if err != nil {
		return "", err
	}

	log.Printf("Request payload: %s", string(bytesPayload))

	req, err := http.NewRequest("POST", defaultURL, bytes.NewReader(bytesPayload))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	log.Printf("Response body: %s", string(body))

	var apiResponse APIResponse
	err = json.Unmarshal(body, &apiResponse)
	if err != nil {
		return "", err
	}

	if len(apiResponse.Output.Choices) > 0 {
		return apiResponse.Output.Choices[0].Text, nil
	}

	return "", fmt.Errorf("no choices found in the response")

}

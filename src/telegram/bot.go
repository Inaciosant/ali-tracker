package telegram

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const defaultTelegramBaseURL = "https://api.telegram.org"

type Bot struct {
	token      string
	baseURL    string
	httpClient *http.Client
}

func NewBot(token string) *Bot {
	return &Bot{
		token:   strings.TrimSpace(token),
		baseURL: defaultTelegramBaseURL,
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

func (b *Bot) SendMessage(chatID, message string) error {
	if b == nil {
		return fmt.Errorf("bot is nil")
	}
	if strings.TrimSpace(b.token) == "" {
		return fmt.Errorf("telegram token is empty")
	}
	if strings.TrimSpace(chatID) == "" {
		return fmt.Errorf("telegram chat id is empty")
	}
	sanitizedMessage := strings.ToValidUTF8(message, "")
	if strings.TrimSpace(sanitizedMessage) == "" {
		return fmt.Errorf("message is empty")
	}

	apiURL := fmt.Sprintf("%s/bot%s/sendMessage", b.baseURL, b.token)
	payload := url.Values{}
	payload.Set("chat_id", chatID)
	payload.Set("text", sanitizedMessage)

	resp, err := b.httpClient.PostForm(apiURL, payload)
	if err != nil {
		return fmt.Errorf("failed to send request to telegram: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("telegram returned %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	return nil
}

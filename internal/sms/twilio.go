package sms

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Client struct {
	accountSID string
	authToken  string
	fromNumber string
	httpClient *http.Client
}

func NewClient(accountSID, authToken, fromNumber string) *Client {
	return &Client{
		accountSID: accountSID,
		authToken:  authToken,
		fromNumber: fromNumber,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

type InboundMessage struct {
	From string
	To   string
	Body string
}

func ParseInbound(r *http.Request) (*InboundMessage, error) {
	if err := r.ParseForm(); err != nil {
		return nil, fmt.Errorf("failed to parse form: %w", err)
	}

	from := r.FormValue("From")
	to := r.FormValue("To")
	body := r.FormValue("Body")

	if from == "" {
		return nil, fmt.Errorf("missing From field")
	}

	return &InboundMessage{
		From: from,
		To:   to,
		Body: body,
	}, nil
}

func (c *Client) Send(to, body string) error {
	apiURL := fmt.Sprintf("https://api.twilio.com/2010-04-01/Accounts/%s/Messages.json", c.accountSID)

	data := url.Values{}
	data.Set("To", to)
	data.Set("From", c.fromNumber)
	data.Set("Body", body)

	req, err := http.NewRequest("POST", apiURL, strings.NewReader(data.Encode()))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.SetBasicAuth(c.accountSID, c.authToken)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		var errResp struct {
			Message string `json:"message"`
			Code    int    `json:"code"`
		}
		if err := json.Unmarshal(respBody, &errResp); err == nil && errResp.Message != "" {
			return fmt.Errorf("twilio error %d: %s", errResp.Code, errResp.Message)
		}
		return fmt.Errorf("twilio returned status %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}

func WriteTwiMLResponse(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/xml")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("<?xml version=\"1.0\" encoding=\"UTF-8\"?><Response></Response>"))
}

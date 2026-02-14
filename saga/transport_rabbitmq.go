package saga

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const DefaultExchange = "saga.events"

type RabbitMQTransport struct {
	client   *http.Client
	endpoint string
	username string
	password string
	vhost    string
	exchange string
}

type rabbitPublishPayload struct {
	Properties      map[string]any `json:"properties"`
	RoutingKey      string         `json:"routing_key"`
	Payload         string         `json:"payload"`
	PayloadEncoding string         `json:"payload_encoding"`
}

type rabbitPublishResponse struct {
	Routed bool `json:"routed"`
}

func NewRabbitMQTransport(endpoint, username, password, vhost, exchange string) (*RabbitMQTransport, error) {
	if endpoint == "" {
		return nil, fmt.Errorf("rabbitmq endpoint is required")
	}
	if exchange == "" {
		exchange = DefaultExchange
	}
	if vhost == "" {
		vhost = "/"
	}
	transport := &RabbitMQTransport{
		client:   &http.Client{Timeout: 5 * time.Second},
		endpoint: strings.TrimRight(endpoint, "/"),
		username: username,
		password: password,
		vhost:    vhost,
		exchange: exchange,
	}

	if err := transport.ensureExchange(); err != nil {
		return nil, err
	}
	return transport, nil
}

func (t *RabbitMQTransport) ensureExchange() error {
	vhostEscaped := url.PathEscape(t.vhost)
	exchangeEscaped := url.PathEscape(t.exchange)
	declareURL := fmt.Sprintf("%s/api/exchanges/%s/%s", t.endpoint, vhostEscaped, exchangeEscaped)
	body := []byte(`{"type":"topic","durable":true,"auto_delete":false,"internal":false,"arguments":{}}`)
	req, err := http.NewRequest(http.MethodPut, declareURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create exchange declare request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if t.username != "" {
		req.SetBasicAuth(t.username, t.password)
	}
	resp, err := t.client.Do(req)
	if err != nil {
		return fmt.Errorf("declare exchange request failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("declare exchange failed with status %d", resp.StatusCode)
	}
	return nil
}

func (t *RabbitMQTransport) Publish(event Event) error {
	body, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal event: %w", err)
	}

	requestPayload := rabbitPublishPayload{
		Properties:      map[string]any{},
		RoutingKey:      event.Name,
		Payload:         string(body),
		PayloadEncoding: "string",
	}
	requestBody, err := json.Marshal(requestPayload)
	if err != nil {
		return fmt.Errorf("marshal rabbitmq payload: %w", err)
	}

	vhostEscaped := url.PathEscape(t.vhost)
	exchangeEscaped := url.PathEscape(t.exchange)
	publishURL := fmt.Sprintf("%s/api/exchanges/%s/%s/publish", t.endpoint, vhostEscaped, exchangeEscaped)

	req, err := http.NewRequest(http.MethodPost, publishURL, bytes.NewReader(requestBody))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if t.username != "" {
		req.SetBasicAuth(t.username, t.password)
	}

	resp, err := t.client.Do(req)
	if err != nil {
		return fmt.Errorf("publish request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return fmt.Errorf("rabbitmq management API returned status %d", resp.StatusCode)
	}

	var parsed rabbitPublishResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return fmt.Errorf("decode publish response: %w", err)
	}
	if !parsed.Routed {
		return fmt.Errorf("event was not routed by rabbitmq")
	}
	return nil
}

func (t *RabbitMQTransport) Close() error {
	return nil
}

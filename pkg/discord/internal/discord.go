package internal

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/websocket"
)

type Service struct {
	// Connect to Bot
	Events chan GatewayEvent

	// technical stuff
	httpClient    *http.Client
	applicationId string
	conn          *websocket.Conn

	// Discord related stuff
	lastSequence  int64
	stopHeartbeat chan struct{}

	// Discord related but has business value? Idk...
	voiceStates map[string]VoiceState
}

func New() *Service {
	newDiscord := &Service{
		httpClient:    &http.Client{},
		applicationId: os.Getenv("APPLICATIONID"),
		Events:        make(chan GatewayEvent),
		stopHeartbeat: make(chan struct{}),
		voiceStates:   make(map[string]VoiceState),
	}

	go newDiscord.startup()

	return newDiscord
}

func (s *Service) startup() {
	gatewayUrl, err := s.getGatewayUrl()
	if err != nil {
		log.Fatalln("Unable to optain gateway URL: ", err)
	}

	log.Println("Dialing: ", gatewayUrl)
	conn, _, err := websocket.DefaultDialer.Dial(gatewayUrl, nil)
	if err != nil {
		log.Fatalln("Unable to connect to gateway: ", err)
	}

	s.conn = conn

	go s.listen()
}

func (s *Service) listen() error {
	defer s.conn.Close()

	for {
		_, eventBytes, err := s.conn.ReadMessage()
		if err != nil {
			// TODO: Reconnect
			return err
		}

		var event GatewayEvent
		if err := json.Unmarshal(eventBytes, &event); err != nil {
			slog.Error("Error parsing Gateway Event:", "Error", err)
			continue
		}

		if event.S != nil {
			s.lastSequence = *event.S
		}

		go s.processEvent(&event)
	}
}

func (s *Service) processEvent(event *GatewayEvent) error {
	slog.Debug("processing Event: ", "OP Code", event.OP)

	switch event.OP {
	case GatewayEventTypeDispatch:
		slog.Debug("Received Dispatch Event.", "Event Name", *event.T)
		switch *event.T {
		case GatewayEventNameReady:
			slog.Debug("Received Ready Event.")

		}

		s.Events <- *event

	case GatewayEventTypeHeartbeat:
		slog.Debug("Received heartbeat")

	case GatewayEventTypeReconnect:
		slog.Debug("Received Reconnect event")

	case GatewayEventTypeInvalidSession:
		slog.Debug("Received Invalid Session event")

	case GatewayEventTypeHello:
		slog.Debug("Received Hello event")

		var helloEventData HelloEventData
		err := json.Unmarshal(event.D, &helloEventData)
		if err != nil {
			slog.Debug("Error parsing Hello Event Data:", "Error", err)
			return err
		}
		go s.startHeartbeat(helloEventData.HeartbeatInterval)
		s.identify()

	case GatewayEventTypeHeartbeatACK:
		slog.Debug("Received Heartbeat ACK")
	}

	return nil
}

func (s *Service) startHeartbeat(interval int) error {
	slog.Debug("Starting heartbeat", "interval", interval)

	ticker := time.NewTicker(time.Duration(interval) * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			slog.Debug("Sending heartbeat")

			data, err := json.Marshal(s.lastSequence)
			if err != nil {
				slog.Debug("Error marshalling heartbeat payload.", "Error", err)
				return err
			}

			heartbeatPayload := GatewayEvent{
				OP: GatewayEventTypeHeartbeat,
				D:  json.RawMessage(data),
			}

			if err := s.conn.WriteJSON(heartbeatPayload); err != nil {
				slog.Debug("Error sending heartbeat.", "Error", err)
				return err
			}
		case <-s.stopHeartbeat:
			return nil
		}
	}
}

func (s *Service) identify() {
	identifyData := IdentifyData{
		Token: os.Getenv("TOKEN"),
		Properties: IdentifyConnectionProperties{
			OS:      "Windows",
			Browser: "jabba_the_bot",
			Device:  "jabba_the_bot",
		},
		Intents: 641,
	}

	data, err := json.Marshal(identifyData)
	if err != nil {
		log.Println("Error marshalling heartbeat payload:", err)
	}

	heartbeatPayload := GatewayEvent{
		OP: GatewayEventTypeIdentify,
		D:  json.RawMessage(data),
	}

	if err := s.conn.WriteJSON(heartbeatPayload); err != nil {
		log.Printf("Heartbeat error: %v", err)
		return
	}
}

func (s *Service) RegisterCommand(payload ApplicationCommand) error {
	url := fmt.Sprintf("https://discord.com/api/v10/applications/%s/commands", s.applicationId)

	res, err := s.post(payload, url)
	if err != nil {
		return err
	}

	_, _ = io.ReadAll(res.Body)

	err = res.Body.Close()
	if err != nil {
		return err
	}

	switch res.StatusCode {
	case http.StatusBadRequest:
		return errors.New("unable to create Command")
	case http.StatusOK:
		log.Println("Command \"", payload.Name, "\" successfully Registered")
	}

	return nil
}

func (s *Service) CreateInteractionResponse(id string, token string, payload InteractionResponse, withResponse bool) ([]byte, error) {
	url := fmt.Sprintf("https://discord.com/api/v10/interactions/%s/%s/callback", id, token)
	if withResponse {
		url += "?with_response=true"
	}

	statusCode, raw, err := s.doRequest("POST", url, payload)
	if err != nil {
		return nil, err
	}
	if statusCode != http.StatusNoContent && statusCode != http.StatusOK {
		return nil, fmt.Errorf("API Error %d: %s", statusCode, string(raw))
	}
	if !withResponse || len(raw) == 0 {
		return nil, nil
	}

	return raw, nil
}

func (s *Service) PostInteractionResponse(id string, token string, payload InteractionResponse) error {
	_, err := s.CreateInteractionResponse(id, token, payload, false)
	return err
}

func (s *Service) GetOriginalInteractionResponse(token string) (*Message, error) {
	url := fmt.Sprintf("https://discord.com/api/v10/webhooks/%s/%s/messages/@original", s.applicationId, token)

	statusCode, raw, err := s.doRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	if statusCode != http.StatusOK {
		return nil, fmt.Errorf("API Error %d: %s", statusCode, string(raw))
	}

	var message Message
	if err := json.Unmarshal(raw, &message); err != nil {
		return nil, err
	}

	return &message, nil
}

func (s *Service) EditOriginalInteractionResponse(token string, payload WebhookMessageEdit) (*Message, error) {
	url := fmt.Sprintf("https://discord.com/api/v10/webhooks/%s/%s/messages/@original", s.applicationId, token)

	statusCode, raw, err := s.doRequest("PATCH", url, payload)
	if err != nil {
		return nil, err
	}
	if statusCode != http.StatusOK {
		return nil, fmt.Errorf("API Error %d: %s", statusCode, string(raw))
	}

	var message Message
	if err := json.Unmarshal(raw, &message); err != nil {
		return nil, err
	}

	return &message, nil
}

func (s *Service) DeleteOriginalInteractionResponse(token string) error {
	url := fmt.Sprintf("https://discord.com/api/v10/webhooks/%s/%s/messages/@original", s.applicationId, token)

	statusCode, raw, err := s.doRequest("DELETE", url, nil)
	if err != nil {
		return err
	}
	if statusCode != http.StatusNoContent {
		return fmt.Errorf("API Error %d: %s", statusCode, string(raw))
	}

	return nil
}

func (s *Service) CreateFollowupMessage(token string, payload InteractionFollowupMessage) (*Message, error) {
	url := fmt.Sprintf("https://discord.com/api/v10/webhooks/%s/%s", s.applicationId, token)

	statusCode, raw, err := s.doRequest("POST", url, payload)
	if err != nil {
		return nil, err
	}
	if statusCode != http.StatusOK {
		return nil, fmt.Errorf("API Error %d: %s", statusCode, string(raw))
	}

	var message Message
	if err := json.Unmarshal(raw, &message); err != nil {
		return nil, err
	}

	return &message, nil
}

func (s *Service) post(payload interface{}, url string) (*http.Response, error) {
	interactionResponseBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(interactionResponseBytes))
	if err != nil {
		return nil, err
	}

	req.Header.Add("Authorization", "Bot "+os.Getenv("TOKEN"))
	req.Header.Set("Content-Type", "application/json")

	return s.httpClient.Do(req)
}

func (s *Service) doRequest(method string, url string, payload interface{}) (int, []byte, error) {
	var body io.Reader
	if payload != nil {
		data, err := json.Marshal(payload)
		if err != nil {
			return 0, nil, err
		}
		body = bytes.NewBuffer(data)
	}

	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return 0, nil, err
	}

	req.Header.Add("Authorization", "Bot "+os.Getenv("TOKEN"))
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, nil, err
	}

	return resp.StatusCode, raw, nil
}

func (s *Service) getGatewayUrl() (string, error) {
	req, _ := http.NewRequest("GET", "https://discord.com/api/gateway", nil)

	res, err := s.httpClient.Do(req)
	if err != nil {
		return "", err
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return "", err
	}

	err = res.Body.Close()
	if err != nil {
		return "", err
	}

	var responseStruct GatewayResponse
	if err := json.Unmarshal(body, &responseStruct); err != nil {
		return "", err
	}

	return fmt.Sprintf("%s/?v=10&encoding=json", responseStruct.Url), nil
}

func (s *Service) UpdateVoiceState(guildId string, channelId string, selfMute bool, selfDeaf bool) error {
	voiceStateData := VoiceState{
		GuildId:   guildId,
		ChannelId: channelId,
		SelfMute:  selfMute,
		SelfDeaf:  selfDeaf,
	}

	data, err := json.Marshal(voiceStateData)
	if err != nil {
		return fmt.Errorf("error marshalling voice state: %w", err)
	}

	payload := &GatewayEvent{
		OP: GatewayEventTypeVoiceStateUpdate,
		D:  data,
	}

	return s.conn.WriteJSON(payload)
}

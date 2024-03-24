// TODOS:
// - handle errors
// - handle heartbeat ack
// - handle snowflakes ...

package main

import (
	"Jabba_The_Bot/internal/pkg/events"
	opcodes "Jabba_The_Bot/internal/pkg/op_codes"
	"Jabba_The_Bot/internal/pkg/structs"
	"bytes"
	"os"

	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"reflect"
	"time"

	"github.com/gorilla/websocket"
	"github.com/joho/godotenv"
)

// structs

// consts

var seq *int
var conn *websocket.Conn
var client *http.Client
var resumeUrl string

var heartbeat chan<- bool

// state machine
var joinRequested bool = false
var guilds []structs.Guild
var voiceStates map[string]structs.VoiceState

func main() {
	/* f, err := os.OpenFile("dev_log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("error opening file: %v", err)
	}
	defer f.Close()

	log.SetOutput(f) */

	loadDotenv()

	initializeCommands()

	voiceStates = make(map[string]structs.VoiceState)

	wsUrl := getWebsocketUrl()

	conn, _, _ = websocket.DefaultDialer.Dial(wsUrl, nil)
	defer conn.Close()

	log.Println("Initialization Completed")

	for {
		// read messages from ws connection
		_, message, err := conn.ReadMessage()
		if err != nil {
			log.Fatalln("Read here :", err)
			log.Fatalln(message)
		}

		log.Printf("Message recived: \n %s\n", message)

		// parse messages into envelope
		var websocketMessage structs.WebsocketMessage
		err = json.Unmarshal(message, &websocketMessage)
		if err != nil {
			log.Println("parse:", err)
		}

		go processMessag(websocketMessage)
	}
}

func initializeCommands() {
	fmt.Println("Initializing commands")
	var applicationCommands []structs.ApplicationCommand

	url := "https://discord.com/api/v10/applications/" + os.Getenv("APPLICATIONID") + "/commands"

	pingCommand := structs.ApplicationCommand{
		Name:        "ping",
		Type:        1,
		Description: "Ping - Pong",
	}

	joinCommand := structs.ApplicationCommand{
		Name:        "join",
		Type:        1,
		Description: "Jabba joins your voice Channel",
	}

	applicationCommands = append(applicationCommands, pingCommand, joinCommand)

	for _, applicationCommand := range applicationCommands {
		registerCommand(applicationCommand, url)
	}
}

func registerCommand(applicationCommand structs.ApplicationCommand, url string) {
	applicationCommandBytes, err := json.Marshal(applicationCommand)
	if err != nil {
		log.Fatalln("Parsing application command:", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(applicationCommandBytes))
	if err != nil {
		log.Fatalln("Creating application command request:", err)
	}

	req.Header.Add("Authorization", "Bot "+os.Getenv("TOKEN"))

	req.Header.Set("Content-Type", "application/json")

	client = &http.Client{}

	res, err := client.Do(req)
	if err != nil {
		log.Fatalln(err)
	}

	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		log.Fatalln(err)
	}

	switch res.StatusCode {
	case http.StatusBadRequest:
		log.Fatalln(
			"unable to create Command", applicationCommand.Name, "\n",
			"Error: ", string(body),
		)
	case http.StatusOK:
		log.Println("Command \"", applicationCommand.Name, "\" successfully Registered")
	}
}

func loadDotenv() {
	err := godotenv.Load()
	if err != nil {
		log.Fatalln("Unable to load config")
	}
}

func processMessag(websocketMessage structs.WebsocketMessage) {
	log.Println("Processing message. OPCODE: ", websocketMessage.OP)

	switch websocketMessage.OP {
	case opcodes.Hello:
		log.Println("Processing Hello event")
		var websocketData structs.HelloData
		json.Unmarshal(*websocketMessage.D, &websocketData)

		heartbeat = setInterval(sendHeartbeat, time.Duration(time.Duration(websocketData.Heartbeat_interval)*time.Millisecond))

		identify()
	case opcodes.Dispatch:
		log.Println("Event", *websocketMessage.T, "recived")

		switch *websocketMessage.T {
		case events.Ready:
			log.Println("Processing Ready event")
			var readyData structs.ReadyData
			json.Unmarshal(*websocketMessage.D, &readyData)

		case events.GuildCreated:
			var guild structs.Guild
			err := json.Unmarshal(*websocketMessage.D, &guild)
			if err != nil {
				log.Fatalln("Error parsing Guild Creation Data: ", err)
			}

			for _, voiceState := range guild.VoiceStates {
				voiceStates[voiceState.UserId] = voiceState
			}

			guilds = append(guilds, guild)

		case events.InteractionCreate:
			processInteraction(websocketMessage)

		case events.VoiceStateUpdate:
			var voiceState structs.VoiceState
			err := json.Unmarshal(*websocketMessage.D, &voiceState)
			if err != nil {
				log.Fatalln("Error parsing voice state information", err)
			}

			voiceStates[voiceState.UserId] = voiceState

			prettyPrintMap(voiceStates)
		}
	}
}

func prettyPrintMap(m any) {
	v := reflect.ValueOf(m)
	if v.Kind() != reflect.Map {
		fmt.Println("not a map!")
		return
	}

	iter := v.MapRange()

	for iter.Next() {
		fmt.Println(iter.Key(), ": ", iter.Value())
	}
}

func processInteraction(websocketMessage structs.WebsocketMessage) bool {
	var err error
	var url string
	var interactionResponseDataJson []byte
	var interactionResponseType int

	log.Println("Processing Interaction")

	fmt.Printf("%s", *websocketMessage.D)

	var interactionData structs.InteractionCreateData
	json.Unmarshal(*websocketMessage.D, &interactionData)

	switch interactionData.Data.Name {
	case "ping":
		interactionResponseType = 4

		interactionResponseData := structs.InteractionCallbackDataMessage{
			TTS:     false,
			Content: "Pong",
			Embeds:  []structs.Embed{},
			AllowMentions: structs.AllowMentions{
				Parse: []string{},
			},
		}

		interactionResponseDataJson, err = json.Marshal(interactionResponseData)
		if err != nil {
			log.Fatalln(err)
		}

	case "join":
		UserVoiceState, ok := voiceStates[interactionData.Member.User.Id]

		if !ok {
			interactionResponseData := structs.InteractionCallbackDataMessage{
				TTS:     false,
				Content: "User not in Voice Channel. Connet to a voice Channel first.",
				Embeds:  []structs.Embed{},
				AllowMentions: structs.AllowMentions{
					Parse: []string{},
				},
			}

			interactionResponseDataJson, err = json.Marshal(interactionResponseData)
			if err != nil {
				log.Fatalln(err)
			}
		}

		voiceStateData := structs.VoiceState{
			GuildId:   interactionData.GuildId,
			ChannelId: UserVoiceState.ChannelId,
			SelfMute:  false,
			SelfDeaf:  false,
		}

		voiceStateDataJson, err := json.Marshal(voiceStateData)
		if err != nil {
			log.Println("parse", err)
		}

		voiceStateDataJsonRaw := json.RawMessage(voiceStateDataJson)

		payload := structs.WebsocketMessage{
			OP: opcodes.VoiceStateUpdate,
			D:  &voiceStateDataJsonRaw,
			S:  nil,
			T:  nil,
		}

		prettyPrint(payload)

		err = conn.WriteJSON(payload)
		if err != nil {
			log.Fatalln(err)
		}

		joinRequested = true

		log.Println("Voice channel Join requestet")
	}

	if len(interactionResponseDataJson) == 0 {
		return true

		interactionResponseType = 4

		interactionResponseData := structs.InteractionCallbackDataMessage{
			TTS:     false,
			Content: "Not implemented yet ...",
			Embeds:  []structs.Embed{},
			AllowMentions: structs.AllowMentions{
				Parse: []string{},
			},
		}

		interactionResponseDataJson, err = json.Marshal(interactionResponseData)
		if err != nil {
			log.Fatalln(err)
		}
	}

	interactionResponse := structs.InteractionResponse{
		Type: interactionResponseType,
		Data: json.RawMessage(interactionResponseDataJson),
	}

	url = fmt.Sprintf(
		"https://discord.com/api/v10/interactions/%s/%s/callback",
		interactionData.Id,
		interactionData.Token,
	)

	interactionResponseBytes, err := json.Marshal(interactionResponse)
	if err != nil {
		log.Fatalln("Parsing application command:", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(interactionResponseBytes))
	if err != nil {
		log.Fatalln("Creating application command request:", err)
	}

	req.Header.Set("Content-Type", "application/json")

	res, err := client.Do(req)
	if err != nil {
		log.Fatalln(err)
	}

	defer res.Body.Close()

	log.Println("Interaction Response sent")
	return false
}

func identify() {
	log.Println("Identification started")

	identifyData := structs.IdentifyData{
		Token: os.Getenv("TOKEN"),
		Properties: structs.IdentifyConnectionProperties{
			OS:      "linux",
			Browser: "jabba_the_bot",
			Device:  "jabba_the_bot",
		},
		Compress:        nil,
		Large_threshold: nil,
		Shard:           nil,
		Presence:        nil,
		Intents:         641,
	}

	bytesIdentifyData, err := json.Marshal(identifyData)
	if err != nil {
		log.Println("parse", err)
	}

	rawIdentifyData := json.RawMessage(bytesIdentifyData)

	payload := structs.WebsocketMessage{
		OP: opcodes.Identify,
		D:  &rawIdentifyData,
		S:  nil,
		T:  nil,
	}

	conn.WriteJSON(payload)

	log.Println("Identification finished")
}

func getWebsocketUrl() string {
	const baseUrl = "https://discord.com/api"
	response, err := http.Get(baseUrl + "/gateway")
	if err != nil {
		log.Fatalln(err)
	}
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		log.Fatalln(err)
	}

	var responseStruct structs.GatewayResponse
	if err != json.Unmarshal(body, &responseStruct) {
		log.Fatalln(err)
	}

	return responseStruct.Url + "/?v=10&encoding=json"
}

func sendHeartbeat() {
	err := conn.WriteJSON(structs.Heartbeat{OP: opcodes.Heartbeat, D: seq})
	if err != nil {
		log.Println("heartbeat:", err)
	}
}

// helper
func setInterval(p any, interval time.Duration) chan<- bool {
	ticker := time.NewTicker(interval)
	stopIt := make(chan bool)

	go func() {
		for {
			select {
			case <-stopIt:
				fmt.Println("stop setInterval")
				return

			case <-ticker.C:
				reflect.ValueOf(p).Call([]reflect.Value{})
			}
		}
	}()

	return stopIt
}

func prettyPrint(data any) {
	empJosn, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		log.Fatalln("error parsing readyData to string", err)
	}

	fmt.Println(string(empJosn))
}

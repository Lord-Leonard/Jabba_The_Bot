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

func main() {
	loadDotenv()

	fmt.Println("Initializing commands")

	url := "https://discord.com/api/v10/applications/" + os.Getenv("APPLICATIONID") + "/commands"

	log.Println(url)

	applicationCommand := structs.ApplicationCommand{
		Name:        "ping",
		Type:        1,
		Description: "Ping - Pong",
	}

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

	fmt.Println(req)

	client = &http.Client{}

	res, err := client.Do(req)
	if err != nil {
		log.Fatalln(err)
	}

	log.Println(res)
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		log.Fatalln(err)
		return
	}

	fmt.Println(string(body))

	wsUrl := getWebsocketUrl()

	// TODO: handle error
	conn, _, _ = websocket.DefaultDialer.Dial(wsUrl, nil)
	defer conn.Close()

	log.Println("Initialization Completed")

	for {
		// read messages from ws
		_, message, err := conn.ReadMessage()
		if err != nil {
			log.Fatalln("Read here :", err)
			log.Fatalln(message)
			return
		}
		log.Println("Data recived")

		// parse messages into envelope
		var websocketMessage structs.WebsocketMessage
		err = json.Unmarshal(message, &websocketMessage)
		if err != nil {
			log.Println("parse:", err)
		}

		processMessag(websocketMessage)
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

		case events.InteractionCreate:
			log.Println("Processing Interaction")

			var interactionData structs.InteractionCreateData
			json.Unmarshal(*websocketMessage.D, &interactionData)

			prettyPrint(interactionData)

			interactionResponseData := structs.InteractionCallbackDataMessage{
				TTS:     false,
				Content: "Pong",
				Embeds:  []structs.Embed{},
				AllowMentions: structs.AllowMentions{
					Parse: []string{},
				},
			}

			interactionResponseDataJson, err := json.Marshal(interactionResponseData)
			if err != nil {
				log.Fatalln(err)
			}

			interactionResponse := structs.InteractionResponse{
				Type: 4,
				Data: json.RawMessage(interactionResponseDataJson),
			}

			url := fmt.Sprintf(
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

			fmt.Println(req)

			res, err := client.Do(req)
			if err != nil {
				log.Fatalln(err)
			}

			log.Println(res)
			defer res.Body.Close()

			log.Println("Interaction Response sent")
		}
	}
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

	return responseStruct.URL + "/?v=10&encoding=json"
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

func prettyPrint(readyData any) {
	empJosn, err := json.MarshalIndent(readyData, "", "  ")
	if err != nil {
		log.Fatalln("error parsing readyData to string", err)
	}

	log.Println(string(empJosn))
}

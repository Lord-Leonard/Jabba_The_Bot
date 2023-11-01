// TODOS:
// - handle errors
// - handle heartbeat ack

package main

import (
	"Jabba_The_Bot/internal/pkg/events"
	opcodes "Jabba_The_Bot/internal/pkg/op_codes"

	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"reflect"
	"time"

	"github.com/gorilla/websocket"
)

// structs
type GatewayResponse struct {
	URL string `json:"url"`
}

type WebsocketMessage struct {
	OP int              `json:"op"`
	D  *json.RawMessage `json:"d"`
	S  *int             `json:"s,omitempty"`
	T  *string          `json:"t,omitempty"`
}

type HelloData struct {
	Heartbeat_interval int `json:"heartbeat_interval"`
}

type IdentifyData struct {
	Token           string                       `json:"token"`
	Properties      IdentifyConnectionProperties `json:"properties"`
	Compress        *bool                        `json:"compress,omitempty"`
	Large_threshold *int                         `json:"large_threshold,omitempty"`
	Shard           *[2]int                      `json:"shard,omitempty"`
	Presence        *json.RawMessage             `json:"presence,omitempty"`
	Intents         int                          `json:"intents"`
}

type IdentifyConnectionProperties struct {
	OS      string `json:"os"`
	Browser string `json:"browser"`
	Device  string `json:"device"`
}

type ReadyData struct {
	V int `json:"v"`
	User
}

type Heartbeat struct {
	OP int  `json:"op"`
	D  *int `json:"d"`
}

type User struct {
}

// consts
const token = "ODkwODM1NjQyNTEzMjQ4MjY2.G9r2oA.pGgf6Xq0I22HhVfpbW6ewExZ9D9VTzcgKY-r9E"

// variables
var seq *int
var c *websocket.Conn
var heartbeat chan<- bool

func main() {
	wsUrl := getWebsocketUrl()

	// TODO: handle error
	c, _, _ = websocket.DefaultDialer.Dial(wsUrl, nil)
	defer c.Close()
	log.Println("connected to Websocket")

	for {
		// read messages from ws
		_, message, err := c.ReadMessage()
		if err != nil {
			log.Println("read:", err)
			return
		}
		log.Println("message recived:", string(message))

		// parse messages into envelope
		var websocketMessage WebsocketMessage
		err = json.Unmarshal(message, &websocketMessage)
		if err != nil {
			log.Println("parse:", err)
		}

		processMessag(websocketMessage)
	}

}

func processMessag(websocketMessage WebsocketMessage) {
	switch websocketMessage.OP {
	case opcodes.Hello:
		var websocketData HelloData
		json.Unmarshal(*websocketMessage.D, &websocketData)

		fmt.Println(websocketData)

		heartbeat = setInterval(sendHeartbeat, time.Duration(time.Duration(websocketData.Heartbeat_interval)*time.Millisecond))

		log.Println("Start identification")
		identify()
	case opcodes.Dispatch:
		switch *websocketMessage.T {
		case events.Ready:

		}
	}
}

func identify() {
	identifyData := IdentifyData{token,
		IdentifyConnectionProperties{"linux", "jabba_the_bot", "jabba_the_bot"},
		nil,
		nil,
		nil,
		nil,
		641,
	}

	bytesIdentifyData, err := json.Marshal(identifyData)
	if err != nil {
		log.Println("parse", err)
	}

	rawIdentifyData := json.RawMessage(bytesIdentifyData)

	payload := WebsocketMessage{opcodes.Identify, &rawIdentifyData, nil, nil}

	c.WriteJSON(payload)
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

	var responseStruct GatewayResponse
	if err != json.Unmarshal(body, &responseStruct) {
		log.Fatalln(err)
	}

	return responseStruct.URL + "/?v=10&encoding=json"
}

func sendHeartbeat() {
	err := c.WriteJSON(Heartbeat{opcodes.Heartbeat, seq})
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

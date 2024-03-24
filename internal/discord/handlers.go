package discord

import (
	"Jabba_The_Bot/internal/ws"

	"encoding/json"
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

func (b *Bot) ProcessInteraction(msg ws.Message, conn *websocket.Conn, httpClient *http.Client) bool {
	//var err error
	var _ string
	var _ []byte
	var _ int

	log.Println("Processing Interaction")

	var interactionData internal.Interaction
	json.Unmarshal(msg.D, &interactionData)

	switch interactionData.Data.Name {
	//case "join":
	//	userVoiceState, ok := b.VoiceStates[interactionData.Member.User.Id]
	//
	//	if !ok {
	//		interactionResponseData := discord.InteractionCallbackDataMessage{
	//			TTS:     false,
	//			Content: "User not in Voice Channel. Connect to a voice Channel first.",
	//			Embeds:  []discord.Embed{},
	//			AllowMentions: discord.AllowMentions{
	//				Parse: []string{},
	//			},
	//		}
	//
	//		_, err = json.Marshal(interactionResponseData)
	//		if err != nil {
	//			log.Fatalln(err)
	//		}
	//	} else {
	//		voiceStateData := discord.VoiceState{
	//			guildId:   interactionData.guildId,
	//			ChannelId: userVoiceState.ChannelId,
	//			SelfMute:  false,
	//			SelfDeaf:  false,
	//		}
	//
	//		payload := struct {
	//			OP int         `json:"op"`
	//			D  interface{} `json:"d"`
	//		}{
	//			OP: 4,
	//			D:  voiceStateData,
	//		}
	//
	//		err = conn.WriteJSON(payload)
	//		if err != nil {
	//			log.Fatalln(err)
	//		}
	//
	//		b.JoinRequested = true
	//		log.Println("Voice channel Join requested")
	//	}

	//case "play":
	//	query := ""
	//	for _, opt := range interactionData.Data.Options {
	//		if opt.Name == "query" {
	//			json.Unmarshal(opt.Value, &query)
	//		}
	//	}
	//
	//	if query == "" {
	//		interactionResponseData := discord.InteractionCallbackDataMessage{
	//			Content: "No query provided.",
	//		}
	//		_, _ = json.Marshal(interactionResponseData)
	//		_ = 4
	//		break
	//	}
	//
	//	userVoiceState, ok := b.VoiceStates[interactionData.Member.User.Id]
	//	if !ok {
	//		interactionResponseData := discord.InteractionCallbackDataMessage{
	//			Content: "You must be in a voice channel to use this command.",
	//		}
	//		_, _ = json.Marshal(interactionResponseData)
	//		_ = 4
	//		break
	//	}
	//
	//	// First join the voice channel if not already there
	//	voiceStateData := discord.VoiceState{
	//		guildId:   interactionData.guildId,
	//		ChannelId: userVoiceState.ChannelId,
	//		SelfMute:  false,
	//		SelfDeaf:  false,
	//	}
	//
	//	payload := struct {
	//		OP int         `json:"op"`
	//		D  interface{} `json:"d"`
	//	}{
	//		OP: 4,
	//		D:  voiceStateData,
	//	}
	//
	//	err = conn.WriteJSON(payload)
	//	if err != nil {
	//		log.Println("Error sending VoiceStateUpdate:", err)
	//	}
	//
	//	b.JoinRequested = true
	//
	//	interactionResponseData := discord.InteractionCallbackDataMessage{
	//		Content: fmt.Sprintf("Searching for: %s", query),
	//	}
	//	_, _ = json.Marshal(interactionResponseData)
	//	_ = 4
	//
	//	// TODO: Implement the actual play logic (search YouTube, connect to voice gateway, stream)
	//	log.Printf("Play requested: %s in guild %s\n", query, interactionData.guildId)
	//
	//	b.Query = query
	//
	//	//if b.Query != "" {
	//	//	vConn.PlayYouTube(b.Query)
	//	//	b.Query = ""
	//	//}
	}

	return false
}

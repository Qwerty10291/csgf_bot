package main

import (
	"github.com/Qwerty10291/csgf_bot/client"
	"github.com/Qwerty10291/csgf_bot/chat"
)

func main() {
	csgfClient := client.NewClient(client.ClientConfig{
		VkLogin:    "login",
		VkPassword: "password",
	})

	chat.NewMathChatGame(csgfClient, 0.05)
	err := csgfClient.Connect()
	if err != nil{
		panic(err)
	}

	csgfClient.StartListener()
}
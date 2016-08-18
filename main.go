package main

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/bwmarrin/discordgo"
)

var session *discordgo.Session
var player *Player
var bot *Bot
var settings *Settings

func main() {
	sigs := make(chan os.Signal, 1)
	done := make(chan bool, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	bot = &Bot{}
	go bot.Init()

	<-done
}

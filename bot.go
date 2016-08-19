package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

type Bot struct {
	ID string
}

func (b *Bot) Init() {
	b.Log("Starting bot")

	set, e := LoadSettings()
	if e == nil {
		settings = &set

		discord, err := discordgo.New(settings.BotToken)
		if err == nil {
			session = discord

			u, _ := session.User("@me")
			b.ID = u.ID

			session.AddHandler(b.MessageCreate)
			session.Open()

			//setup player
			vc, vce := session.Channel(settings.PlayerVoiceChannel)
			if vce == nil {
				player = &Player{}
				pe := player.Init(vc.GuildID, vc.ID)
				if pe != nil {
					bot.Log("Player init failed: " + pe.Error())
				}
				
				test := "https://www.youtube.com/watch?v=9EECxMG-w0M"
				n, u := GetYTAudio(test)
				if u != "" {
					player.AddSong(n, u)
				}
			} else {
				b.Log("Error getting voice channel, player was not setup")
			}
		}
	}
}

func (b *Bot) MessageCreate(s *discordgo.Session, msg *discordgo.MessageCreate) {

	if msg.Author.ID == b.ID {
		return
	}

	if msg.Content[0] == '!' {
		//handle command
		var cmd string
		if strings.Contains(msg.Content, " ") {
			cmd = msg.Content[0:strings.Index(msg.Content, " ")]
		} else {
			cmd = msg.Content
		}

		b.Log("Got command: " + cmd + " from " + msg.Author.Username)

		switch cmd {
		case "!ping":
			{
				s.ChannelMessageSend(msg.ChannelID, "PONG")
				break
			}
		case "!speak":
			{
				v := msg.Content[strings.Index(msg.Content, " "):]
				s.ChannelMessageSendTTS(msg.ChannelID, v)
				break
			}
		case "!play":
			{
				v := strings.Split(msg.Content, " ")[1]
				n, u := GetYTAudio(v)
				if u != "" {
					s.ChannelMessageSend(msg.ChannelID, fmt.Sprintf("%s has been added to the queue", n))
					player.AddSong(n, u)
				} else {
					s.ChannelMessageSend(msg.ChannelID, "Nothing found for "+v)
				}
				break
			}
		case "!info":
			{
				v := strings.Split(msg.Content, " ")[1]

				det := GetYTInfo(v)
				var m []string
				for _, z := range det.Formats {
					m = append(m, fmt.Sprintf("%s - %s", z.Format, z.Protocol))
				}
				s.ChannelMessageSend(msg.ChannelID, strings.Join(m, "\n"))
				break
			}
		case "!pause":
			{
				player.Pause()
				break
			}
		case "!queue":
			{
				x := 1
				for v := range player.playlist {
					s.ChannelMessageSend(msg.ChannelID, fmt.Sprintf("%d: %s", x, v.name))
					x++
				}
				break
			}
		}
		s.ChannelMessageDelete(msg.ChannelID, msg.ID)
	}
}

func (b *Bot) Log(msg string) {
	fmt.Printf("%s: %s\n", time.Now().Format("2006-01-02 15:04:05"), msg)
}

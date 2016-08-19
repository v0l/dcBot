package main

import (
	"encoding/binary"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"strconv"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/layeh/gopus"
)

type Song struct {
	url  string
	name string
}

type Player struct {
	playlist          chan Song
	vc                *discordgo.VoiceConnection
	pcm               chan []int16
	play              chan int
	sampleRate        int
	channels          int
	frameSize         int
	maxBytes          int
	lastPacket        time.Time
	lastSpeakingState bool
	paused            bool
}

func (p *Player) Init(g string, c string) error {
	p.sampleRate = 48000
	p.channels = 2
	p.frameSize = 960
	p.maxBytes = (p.sampleRate * p.channels)

	p.pcm = make(chan []int16, 2)
	p.playlist = make(chan Song, 256)
	p.play = make(chan int, 2)

	p.lastSpeakingState = true //make sure the play state is true when we start the service
	p.lastPacket = time.Now()

	dgv, err := session.ChannelVoiceJoin(g, c, false, false)
	if err == nil {
		p.vc = dgv
		go p.PlayerTick()
		go p.PlayerConvThread()
		go p.PlayerThread()
	}

	return err
}

func (p *Player) PlayerTick() {
	for {
		if time.Now().Sub(p.lastPacket).Nanoseconds() > time.Second.Nanoseconds() {
			if p.lastSpeakingState == true {
				p.vc.Speaking(false)
				bot.Log("Audio has stopped..")
				p.UpdateStatus("")

				//audio has stopped, start next track if its available
				p.play <- 0
			}

			p.lastSpeakingState = false

		} else {
			if p.lastSpeakingState == false {
				p.vc.Speaking(true)
				bot.Log("Audio has started..")
			}

			p.lastSpeakingState = true
		}

		time.Sleep(time.Microsecond)
	}
}

func (p *Player) PlayerThread() {
	opusEncoder, err := gopus.NewEncoder(p.sampleRate, p.channels, gopus.Audio)
	if err == nil {
		for {
			if p.paused {
				time.Sleep(time.Microsecond)
				continue
			}

			pcm := <-p.pcm //grab pcm samples from chan

			opus, err := opusEncoder.Encode(pcm, p.frameSize, p.maxBytes)
			if err == nil {
				if p.vc.Ready || p.vc.OpusSend != nil {
					p.lastPacket = time.Now()
					p.vc.OpusSend <- opus //pipe opus data to channel
				} else {
					bot.Log("Discordgo not ready for opus packets")
				}
			} else {
				bot.Log("Opus encode error: " + err.Error())
			}
		}
	} else {
		bot.Log("Opus encoder failed to init: " + err.Error())
	}
}

func (p *Player) PlayerConvThread() {
	for {
		s := <-p.playlist //take a song from the queue

		bot.Log("Downloading file: " + s.url)
		rsp, er := http.Get(s.url)

		if er == nil {
			p.UpdateStatus(s.name)

			bot.Log("Streaming data: " + s.name)
			run := exec.Command("ffmpeg", "-i", "-", "-f", "s16le", "-ar", strconv.Itoa(p.sampleRate), "-ac", strconv.Itoa(p.channels), "pipe:1")
			ffmpegin, _ := run.StdinPipe()
			ffmpegout, _ := run.StdoutPipe()
			run.Start()

			go io.Copy(ffmpegin, rsp.Body)

			audiobuf := make([]int16, p.frameSize*p.channels)
			for {
				erb := binary.Read(ffmpegout, binary.LittleEndian, &audiobuf)
				if erb == io.EOF || erb == io.ErrUnexpectedEOF {
					break
				}

				p.pcm <- audiobuf
			}

			rsp.Body.Close()
		} else {
			bot.Log("Error downloading song: " + er.Error())
		}
	}
}

func (p *Player) AddSong(name, url string) {
	ns := Song{}
	ns.name = name
	ns.url = url

	p.playlist <- ns
}

func (p *Player) UpdateStatus(s string) {
	if s != "" {
		session.ChannelMessageSend(settings.PlayerTextChannel, fmt.Sprintf("Now playing - %s", s))
	}
	session.UpdateStatus(0, s)
}

func (p *Player) Pause() {
	p.paused = !p.paused
}

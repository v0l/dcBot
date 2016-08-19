package main

import (
	"fmt"
	"io"
	"os/exec"
	"encoding/binary"
	"strconv"
	"time"
	"bufio"
	
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
	lastPacket        time.Time
	lastSpeakingState bool
	paused            bool
}

func (p *Player) Init(g string, c string) error {
	p.sampleRate = 48000
	p.channels = 2
	p.frameSize = 960
	
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

func (p *Player) GetOpusPacketZeroSize(b0 uint32, fs uint32) uint32{
	var audiosize uint32;
	
	if b0&0x80 > 0 {
		audiosize = ((b0 >> 3)&0x3)
		audiosize = (fs << audiosize) / 400
	} else if (b0&0x60) == 0x60 {
		if b0&0x08 > 0 {
			audiosize = fs / 50
		}else {
			audiosize = fs / 100
		}
	} else {
		audiosize = ((b0>>3)&0x3)
		if audiosize == 3 {
			audiosize = fs*60/1000
		}else{
			audiosize = (fs << audiosize)/100
		}
	}
	return audiosize
}

func (p *Player) GetOpusPacketOtherSize(data []byte) uint32 {
	var len uint32
	
	h0 := uint8(data[1])
	h1 := uint8(data[2])

	if h0 > 251 {
		len = (uint32(h1) * 4) + uint32(h0)
	}else{
		len = uint32(h0)
	}
	
	return len
}

func (p *Player) GetOpusPacketLength(data []byte) uint32{
	var len uint32
	
	hz := data[0]&0x3
	
	switch hz {
		case 0:{
			len = p.GetOpusPacketZeroSize(uint32(data[0]), uint32(p.sampleRate)) + 1
			break
		}
		case 1:
		case 2:
		case 3:{
			len = p.GetOpusPacketOtherSize(data)
			if len > 251 {
				len += 2
			}else{
				len += 1
			}
			break
		}
	}
	
	bot.Log(fmt.Sprintf("Type: %d, Length: %d", hz, len))
	return len
}

func (p *Player) PlayerDirect(){
	for {
		s := <-p.playlist //take a song from the queue

		p.UpdateStatus(s.name)

		bot.Log("Streaming data: " + s.name)
	
		run := exec.Command("ffmpeg", "-i", s.url, "-c:a", "libopus", "-vbr", "on" ,"-application", "audio", "-ar", strconv.Itoa(p.sampleRate), "-ac", strconv.Itoa(p.channels), "-f", "opus", "pipe:1")
		ffmpegout, _ := run.StdoutPipe()
		run.Start()
		r := bufio.NewReader(ffmpegout)

		for {
			h, _ := r.Peek(3)
			
			len := int(p.GetOpusPacketLength(h))

			if len == 0{
				break
			}
			
			frame := make([]byte, len)
			rlen, _ := r.Read(frame)
			if rlen == len {
				p.SendAudio(frame)
			}
			frame = nil
		}
	}
}

func (p *Player) PlayerConvThread() {
	for {
		s := <-p.playlist //take a song from the queue

		p.UpdateStatus(s.name)

		bot.Log("Streaming data: " + s.name)
		opus,_ := gopus.NewEncoder(p.sampleRate, p.channels, gopus.Audio)
		opus.SetVbr(true)
		
		run := exec.Command("ffmpeg", "-i", s.url, "-f", "s16le", "-ar", strconv.Itoa(p.sampleRate), "-ac", strconv.Itoa(p.channels), "pipe:1")
		ffmpegout, _ := run.StdoutPipe()
		run.Start()
		
		audiobuf := make([]int16, p.frameSize * 2)
		for {
			er := binary.Read(ffmpegout, binary.LittleEndian, &audiobuf)
			if er == io.EOF || er == io.ErrUnexpectedEOF {
				break
			}
			
			o, _ := opus.Encode(audiobuf, p.frameSize, (p.frameSize * 2) * 2)
			
			p.SendAudio(o)
		}
		
		opus.ResetState()
	}
}

func (p *Player) SendAudio(d []byte){
	if p.vc.Ready || p.vc.OpusSend != nil {
		p.lastPacket = time.Now()
		p.vc.OpusSend <- d //pipe opus data to channel
	} else {
		bot.Log("Discordgo not ready for opus packets")
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

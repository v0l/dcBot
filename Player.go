package main

import (
	"github.com/bwmarrin/discordgo"
	"github.com/layeh/gopus"
	"fmt"
	"net/http"
	"crypto/sha256"
	"io/ioutil"
	"encoding/hex"
	"os/exec"
	"io"
	"os"
	"path/filepath"
	"encoding/binary"
	"time"
	"strconv"
)

type Song struct{
	url string
	name string
	file string
}

type Player struct {
	playlist chan Song
	vc *discordgo.VoiceConnection
	pcm chan []int16
	play chan int
	sampleRate int
	channels int
	frameSize int
	maxBytes int
	lastPacket time.Time
	lastSpeakingState bool
	paused bool
}

func (p * Player) Init(g string, c string) error{
	p.sampleRate = 48000
	p.channels = 2
	p.frameSize = 960
	p.maxBytes = (p.frameSize * 2) * 2
	
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
		if time.Now().Sub(p.lastPacket).Nanoseconds() > time.Second.Nanoseconds(){
			if p.lastSpeakingState == true {
				p.vc.Speaking(false)
				bot.Log("Audio has stopped..")
				p.UpdateStatus("")
				
				//audio has stopped, start next track if its available
				p.play <- 0
			}
			
			p.lastSpeakingState = false
			
		}else{
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
		for{
			if p.paused {
				time.Sleep(time.Microsecond)
				continue
			}
			
			pcm := <- p.pcm //grab pcm samples from chan
			
			opus, err := opusEncoder.Encode(pcm, p.frameSize, p.maxBytes)
			if err == nil {
				if p.vc.Ready || p.vc.OpusSend != nil {
					p.lastPacket = time.Now()
					p.vc.OpusSend <- opus //pipe opus data to channel
				}else{
					bot.Log("Discordgo not ready for opus packets")
				}
			}else{
				bot.Log("Opus encode error: " + err.Error())
			}
		}
	}else{
		bot.Log("Opus encoder failed to init: " + err.Error())
	}
}

func (p *Player) PlayerConvThread(){
	for {
		//<- p.play //block and wait for play signal if no audio is playing
		s := <- p.playlist //take a song from the queue
		so,se := p.DownloadSong(s)
		
		if se == nil {
			p.UpdateStatus(so.name)
			
			bot.Log("Converting file: " + so.file)
			run := exec.Command("ffmpeg", "-i", so.file, "-f", "s16le", "-ar", strconv.Itoa(p.sampleRate), "-ac", strconv.Itoa(p.channels), "pipe:1")
			ffmpegout, _ := run.StdoutPipe()
			run.Start()

			audiobuf := make([]int16, p.frameSize * p.channels)
			for {
				erb := binary.Read(ffmpegout, binary.LittleEndian, &audiobuf)
				if erb == io.EOF || erb == io.ErrUnexpectedEOF {
					break
				}
				
				p.pcm <- audiobuf
			}

			bot.Log("Convert complete..")
		}else {
			bot.Log("Error downloading song: " + se.Error())
		}
	}
}

func (p *Player) AddSong(name, url string) {
	ns := Song{}
	ns.name = name
	ns.url = url
	
	p.playlist <- ns
}

func (p* Player) DownloadSong(song Song) (Song, error) {
	var outer error
	
	rsp, er := http.Get(song.url)
	if er == nil{
		defer rsp.Body.Close()
		
		bot.Log("Downloading file: " + song.url)
		sha := sha256.New()
		
		data, der := ioutil.ReadAll(rsp.Body)
		if der == nil {
			//encode the audio file to opus
			sha.Write(data)
			hash := hex.EncodeToString(sha.Sum(nil))
			
			dir, _ := filepath.Abs(filepath.Dir(os.Args[0]))
			song.file = fmt.Sprintf("%s\\audio\\%s", dir, hash)
			bot.Log("Saving audio file: " + song.file)
			ioutil.WriteFile(song.file, data, 644)
		}else{
			outer = der
		}
	}else{
		outer = er
	}
	
	return song, outer
}

func (p *Player) UpdateStatus(s string) {
	session.ChannelMessageSend(settings.PlayerTextChannel, fmt.Sprintf("Now playing - %s", s))
	session.UpdateStatus(0, s)
}

func (p *Player) Pause(){
	p.paused = !p.paused
}
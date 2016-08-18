package main

import (
	"bytes"
	"encoding/json"
	"os/exec"
)

type youtubedl struct {
	Fulltitle         string      `json:"fulltitle"`
	ViewCount         int         `json:"view_count"`
	URL               string      `json:"url"`
	Extractor         string      `json:"extractor"`
	EndTime           interface{} `json:"end_time"`
	Categories        []string    `json:"categories"`
	FormatNote        string      `json:"format_note"`
	License           string      `json:"license"`
	StartTime         interface{} `json:"start_time"`
	Tbr               float64     `json:"tbr"`
	FormatID          string      `json:"format_id"`
	IsLive            interface{} `json:"is_live"`
	Duration          int         `json:"duration"`
	Ext               string      `json:"ext"`
	Annotations       interface{} `json:"annotations"`
	ExtractorKey      string      `json:"extractor_key"`
	Format            string      `json:"format"`
	Protocol          string      `json:"protocol"`
	WebpageURL        string      `json:"webpage_url"`
	Filesize          int         `json:"filesize"`
	AutomaticCaptions struct {
	} `json:"automatic_captions"`
	Preference         int         `json:"preference"`
	AltTitle           interface{} `json:"alt_title"`
	UploaderID         string      `json:"uploader_id"`
	PlaylistIndex      interface{} `json:"playlist_index"`
	Thumbnail          string      `json:"thumbnail"`
	PlayerURL          string      `json:"player_url"`
	UploaderURL        string      `json:"uploader_url"`
	DislikeCount       int         `json:"dislike_count"`
	AgeLimit           int         `json:"age_limit"`
	AverageRating      float64     `json:"average_rating"`
	UploadDate         string      `json:"upload_date"`
	WebpageURLBasename string      `json:"webpage_url_basename"`
	RequestedSubtitles interface{} `json:"requested_subtitles"`
	ID                 string      `json:"id"`
	DisplayID          string      `json:"display_id"`
	HTTPHeaders        struct {
		Accept         string `json:"Accept"`
		UserAgent      string `json:"User-Agent"`
		AcceptLanguage string `json:"Accept-Language"`
		AcceptEncoding string `json:"Accept-Encoding"`
		AcceptCharset  string `json:"Accept-Charset"`
	} `json:"http_headers"`
	Creator     interface{} `json:"creator"`
	Abr         int         `json:"abr"`
	LikeCount   int         `json:"like_count"`
	Playlist    interface{} `json:"playlist"`
	Filename    string      `json:"_filename"`
	Title       string      `json:"title"`
	Uploader    string      `json:"uploader"`
	Description string      `json:"description"`
	Formats     []struct {
		URL         string  `json:"url"`
		Format      string  `json:"format"`
		FormatNote  string  `json:"format_note"`
		Vcodec      string  `json:"vcodec"`
		Filesize    int     `json:"filesize,omitempty"`
		Abr         int     `json:"abr,omitempty"`
		Tbr         float64 `json:"tbr,omitempty"`
		Acodec      string  `json:"acodec"`
		FormatID    string  `json:"format_id"`
		PlayerURL   string  `json:"player_url"`
		Ext         string  `json:"ext"`
		Protocol    string  `json:"protocol"`
		HTTPHeaders struct {
			Accept         string `json:"Accept"`
			UserAgent      string `json:"User-Agent"`
			AcceptLanguage string `json:"Accept-Language"`
			AcceptEncoding string `json:"Accept-Encoding"`
			AcceptCharset  string `json:"Accept-Charset"`
		} `json:"http_headers"`
		Preference int    `json:"preference,omitempty"`
		Container  string `json:"container,omitempty"`
		Fps        int    `json:"fps,omitempty"`
		Width      int    `json:"width,omitempty"`
		Height     int    `json:"height,omitempty"`
		Resolution string `json:"resolution,omitempty"`
	} `json:"formats"`
	Acodec     string `json:"acodec"`
	Thumbnails []struct {
		URL string `json:"url"`
		ID  string `json:"id"`
	} `json:"thumbnails"`
	Vcodec    string `json:"vcodec"`
	Subtitles struct {
	} `json:"subtitles"`
	Tags []string `json:"tags"`
}

func GetYTInfo(c string) youtubedl {
	cmd := exec.Command("youtube-dl", "-s", "--no-warnings", "--print-json", c)
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err == nil {
		yt := youtubedl{}
		de := json.Unmarshal(out.Bytes(), &yt)
		if de == nil {
			return yt
		}
	}
	return youtubedl{}
}

func GetYTAudio(v string) (string, string) { 
	var outn, outu string
	yt := GetYTInfo(v)
	outn = yt.Title
	
	for _, a := range yt.Formats {
		if a.Acodec == "opus" { //take first opus track we find
			outu = a.URL
			break
		}
	}
	
	if outu == "" {
		for _, a := range yt.Formats {
			if a.Protocol == "https" || a.Protocol == "http" { //take first direct file link
				outu = a.URL
				break
			}
		}
	}
	
	return outn, outu
}
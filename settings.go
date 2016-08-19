package main

import (
	"encoding/json"
	"io/ioutil"
)

type Settings struct {
	PlayerTextChannel  string
	PlayerVoiceChannel string
	BotToken           string
}

func LoadSettings() (Settings, error) {
	var outs Settings
	var oute error

	data, de := ioutil.ReadFile("settings.json")
	if de == nil {
		json.Unmarshal(data, &outs)
	} else {
		oute = de

		//save and empty settings file
		SaveSettings(&Settings{})
	}

	return outs, oute
}

func SaveSettings(s *Settings) {
	data, de := json.MarshalIndent(s, "", "\t")
	if de == nil {
		we := ioutil.WriteFile("settings.json", data, 644)
		if we != nil {
			bot.Log("Error writing settings file: " + we.Error())
		}
	}
}

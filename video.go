package main

import (
	"bytes"
	"encoding/json"
	"os/exec"
)

type ffprobeOutput struct {
	Streams []ffprobeStream `json:"streams"`
}

type ffprobeStream struct {
	Index  int     `json:"index"`
	Width  float32 `json:"width"`
	Height float32 `json:"height"`
}

func getVideoAspectRatio(filePath string) (string, error) {
	cmd := exec.Command("ffprobe", "-v", "error", "-print_format", "json", "-show_streams", filePath)

	buff := bytes.Buffer{}
	cmd.Stdout = &buff

	err := cmd.Run()
	if err != nil {
		return "", err
	}

	output := ffprobeOutput{}
	err = json.Unmarshal(buff.Bytes(), &output)
	if err != nil {
		return "", err
	}

	ratio := output.Streams[0].Width / output.Streams[0].Height
	if ratio > 1.7 && ratio < 1.8 {
		return "16:9", nil
	} else if ratio > 0.5 && ratio < 0.6 {
		return "9:16", nil
	} else {
		return "other", nil
	}
}

package main

import (
	"net/http"
	"sync"

	"fyne.io/fyne"
)

type TMI struct { // TMI Download API 구조체
	Result bool   `json:"result"`
	Data   string `json:"data"`
	Code   int    `json:"code"`
}

type Status struct { // 업데이트 정보 구조체
	Version string `json:"version"`
	NoteKO  string `json:"note_ko"`
	NoteEN  string `json:"note_en"`
}

type TwitchVOD struct { // Twitch v5 Videos Reference - Get Video 구조체
	MutedSegments []struct {
		Duration int `json:"duration"`
		Offset   int `json:"offset"`
	} `json:"muted_segments"`
}

type TwitchOAuth2 struct { // Twitch OAuth2 구조체
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

type TwitchUser struct {
	DisplayName string `json:"display_name"`
	ID          string `json:"_id"`
	Name        string `json:"name"`
	Email       string `json:"email"`
}

type counter struct { // goroutine Counter 구조체
	i  int
	mu sync.Mutex
}

type HumanReadableError interface {
	HumanError() string
	HTTPCode() int
}

type HumanReadableWrapper struct {
	ToHuman string
	Code    int
	error
}

type Handler func(http.ResponseWriter, *http.Request) error

type appInfo struct {
	name string
	icon fyne.Resource
}

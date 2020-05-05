package main

import (
	"net/http"
	"sync"

	"fyne.io/fyne"
)

//TMI TMI Download API 구조체
type TMI struct {
	Result bool   `json:"result"`
	Data   string `json:"data"`
	Code   int    `json:"code"`
}

//Status 업데이트 정보 구조체
type Status struct {
	Version string `json:"version"`
	NoteKO  string `json:"note_ko"`
	NoteEN  string `json:"note_en"`
}

//TwitchVOD Twitch v5 Videos Reference - Get Video 구조체
type TwitchVOD struct {
	MutedSegments []struct {
		Duration int `json:"duration"`
		Offset   int `json:"offset"`
	} `json:"muted_segments"`
}

//TwitchOAuth2 Twitch OAuth2 구조체
type TwitchOAuth2 struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

//TwitchUser Twitch Users 구조체
type TwitchUser struct {
	DisplayName string `json:"display_name"`
	ID          string `json:"_id"`
	Name        string `json:"name"`
	Email       string `json:"email"`
}

//counter goroutine Counter 구조체
type counter struct {
	i  int
	mu sync.Mutex
}

//HumanReadableError Twitch OAuth2
type HumanReadableError interface {
	HumanError() string
	HTTPCode() int
}

//HumanReadableWrapper Twitch OAuth2
type HumanReadableWrapper struct {
	ToHuman string
	Code    int
	error
}

//Handler Twitch OAuth2
type Handler func(http.ResponseWriter, *http.Request) error

//appInfo GUI App 정보
type appInfo struct {
	name string
	icon fyne.Resource
}

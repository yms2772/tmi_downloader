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

//SendLoginInfos 로그인 기록 구조체
type SendLoginInfos struct {
	Type int `json:"type"`
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

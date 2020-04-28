package main

import (
	"os/exec"

	"fyne.io/fyne"
	"fyne.io/fyne/widget"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/gorilla/sessions"
	"github.com/nicklaw5/helix"
	"golang.org/x/oauth2"
)

const ( // OAuth2 Key 상수
	stateCallbackKey = "oauth-state-callback"
	oauthSessionName = "oauth-session"
	oauthTokenKey    = "oauth-token"
)

const ( // API
	allinone = "https://dl.tmi.tips/api/allinone"
)

const (
	B  = 1
	KB = 1024 * B
	MB = 1024 * KB
	GB = 1024 * MB
)

var ( // Main 변수
	version        = "20200427"
	title          = "TMI Downloader"
	dirTemp        = VarOS("dirTemp")
	dirBin         = VarOS("dirBin")
	dirDefDown     = VarOS("dirDefDown")
	dirThumb       = dirTemp + "/thumb"
	fontInfo       = dirBin + "/AppleSDGothicNeoB.ttf"
	ffmpegURL      = VarOS("ffmpegURL")
	ffmpegBinary   = VarOS("ffmpegBinary")
	lang           = setLang()
	chromeStatus   = CheckChrome()
	checkClipboard bool
	err            error
)

var ( // 대기열 변수
	queueID         []string
	queueTitle      []string
	queueTime       []string
	queueThumb      []string
	queueProgress   []*widget.ProgressBar
	queueProgStatus []*widget.Entry
	queueStatus     []*widget.Label
	queueCmd        []*exec.Cmd
)

var ( // Twitch OAuth2 Info
	clientID     = "z0hu5c3qqq1r19om4hfpll7uzirzd0"
	clientSecret = "i6zbcggkyhc970yknht90d7jn0bdfe"
	scopes       = []string{"channel:read:subscriptions", "user:read:email", "user_read", "channel_check_subscription", "user_subscriptions"}
	redirectURL  = "http://localhost:7001/redirect"
	oauth2Config *oauth2.Config
	cookieSecret = []byte("secret")
	cookieStore  = sessions.NewCookieStore(cookieSecret)

	twitchAccessToken  string
	twitchRefreshToken string
	twitchDisplayName  string
	twitchUserName     string
	twitchUserID       string
	twitchUserEmail    string
)

var ( // Function 변수
	helixClient                                          *helix.Client
	button                                               *widget.Button
	intervalCheck, intervalStartCheck, intervalStopCheck *widget.Check
	a                                                    fyne.App
	splWindow, w                                         fyne.Window
	bot                                                  *tgbotapi.BotAPI
)
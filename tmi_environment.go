package main

import (
	"os"

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
	tdownloaderAPI = "https://api.tmi.tips/request/tdownloader"
	loginMemberAPI = "https://api.tmi.tips/request/LoginMember"
	versionAPI     = "https://api.tmi.tips/request/version"
)

var ( // Main 변수
	version        = "0703"
	title          = "TMI Downloader ver." + version
	dirTemp        string
	dirThumb       string
	dirBin         = VarOS("dirBin")
	dirWebFonts    = VarOS("dirWebFonts")
	dirDefDown     = VarOS("dirDefDown")
	ffmpegURL      = VarOS("ffmpegURL")
	ffmpegBinary   = VarOS("ffmpegBinary")
	lang           string
	chromeStatus   = CheckChrome()
	checkClipboard bool
	programUUID    string
	debugFileName  string
	err            error
	errCount       int
)

var ( // 대기열 변수
	queue       = make(map[int]*QueueInfo)
	nowProgress int
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
	debugLog                              *os.File
	helixClient                           *helix.Client
	intervalStartCheck, intervalStopCheck *widget.Check
	keyEntry                              *enterEntry
	a                                     fyne.App
	splWindow, w                          fyne.Window
	bot                                   *tgbotapi.BotAPI
	queueContent                          *widget.Group
	mainContent                           *fyne.Container
)

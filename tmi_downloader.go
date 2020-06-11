package main

import (
	"encoding/gob"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"fyne.io/fyne"
	"fyne.io/fyne/app"
	"fyne.io/fyne/canvas"
	"fyne.io/fyne/dialog"
	"fyne.io/fyne/driver/desktop"
	"fyne.io/fyne/widget"

	tgbot "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/gofrs/uuid"
	"github.com/nicklaw5/helix"
	"github.com/zserge/lorca"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/twitch"
)

func main() { // 메인
	uuid1, err := uuid.NewV4()
	ErrHandle(err)

	programUUID = uuid1.String()
	debugFileName = fmt.Sprintf("%s/debug_%s.txt", dirBin, programUUID)

	debugFiles, err := filepath.Glob(dirBin + "/debug_*")
	ErrHandle(err)

	for _, debugFile := range debugFiles {
		err := os.Remove(debugFile)
		ErrHandle(err)
	}

	ioutil.WriteFile(debugFileName, []byte(fmt.Sprintf(fmt.Sprintf("===== %s 시작\n실행 UUID: %s\n\n", time.Now().Format("2006-01-02 15:04:05"), programUUID))), 0644)

	defer Recover() // 복구

	var updateFlag, resetFlag bool
	var loginFlag string

	flag.BoolVar(&updateFlag, "update", true, "업데이트 확인")
	flag.BoolVar(&resetFlag, "reset", false, "초기화")

	flag.StringVar(&loginFlag, "login", "online", "로그인 모드")

	flag.Parse()

	if resetFlag {
		resetFiles, err := filepath.Glob(dirBin + "/*")
		ErrHandle(err)

		for _, resetFile := range resetFiles {
			err := os.Remove(resetFile)
			ErrHandle(err)
		}

		RunAgain()
	}

	if loginFlag == "logout" {
		a.Preferences().RemoveValue("twitchRefreshToken")

		RunAgain()
	}

	debugLog, err = os.OpenFile(debugFileName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	ErrHandle(err)
	defer debugLog.Close()

	logoImage := &canvas.Image{
		Resource: logo,
		FillMode: canvas.ImageFillOriginal,
	}
	canvas.Refresh(logoImage)
	logoImage.Resize(fyne.NewSize(50, 50))

	a = app.NewWithID("tmi.tips.dl")

	lang = a.Preferences().StringWithFallback("language", "Korean")

	appInfo := &appInfo{
		name: "TMI Downloader",
	}

	appInfo.icon = logo

	a.SetIcon(appInfo.icon)
	a.Settings().SetTheme(NewCustomTheme())

	drv := a.Driver().(desktop.Driver)

	splWindow = drv.CreateSplashWindow()
	splWindow.SetTitle(title)
	splWindow.Resize(fyne.NewSize(300, 200))
	splWindow.SetFixedSize(true)
	splWindow.CenterOnScreen()

	w = a.NewWindow(title)
	w.CenterOnScreen()
	w.SetFixedSize(true)
	w.SetIcon(appInfo.icon)

	if loginFlag == "offline" {
		title = "TMI Downloader Offline Mode"
	}

	err = os.MkdirAll(dirBin, 0777)
	ErrHandle(err)

	err = os.MkdirAll(dirWebFonts, 0777)
	ErrHandle(err)

	err = os.MkdirAll(dirThumb, 0777)
	ErrHandle(err)

	fmt.Println("언어: " + LoadLang("lang"))

	//w.Canvas().SetOnTypedKey(func(k *fyne.KeyEvent) { // 키 이벤트
	//	if k.Name == fyne.KeyQ { // q 눌러서 종료
	//
	//	}
	//})

	w.SetOnClosed(func() { // 강제 종료
		if len(queueCmd) != 0 {
			fmt.Println("Quit: Forced")
			for _, cmdProgress := range queueCmd {
				err = cmdProgress.Process.Kill()
				ErrHandle(err)
			}
		}

		ClearDir(dirTemp)

		os.Exit(0)
	})

	w.SetTitle(title)
	w.SetContent(DownloadHome(w))

	go func() {
		_, noFFmpeg := os.Stat(dirBin + "/" + ffmpegBinary)

		if os.IsNotExist(noFFmpeg) {
			splWindow.SetContent(SplBox(LoadLang("downloadNecessary"), logoImage))

			if _, err := os.Stat(dirBin + "/" + ffmpegBinary); os.IsNotExist(err) {
				Download(dirBin+`/ffmpeg.tar.gz`, ffmpegURL) // ffmpeg 다운로드

				r, err := os.Open(dirBin + "/ffmpeg.tar.gz")
				ErrHandle(err)
				defer r.Close()

				err = Untar(dirBin, r)
				ErrHandle(err)
			}

			splWindow.SetContent(SplBox(LoadLang("downloadComplete"), logoImage))
		}

		splWindow.SetContent(SplBox(LoadLang("loadProgram"), logoImage))

		needUpdate, updateNote := CheckUpdate()
		if !updateFlag {
			needUpdate = false
		}

		bot, _ = tgbot.NewBotAPI("1267111133:AAEyfJ66CNHH956wT-efPrXnFiNAVVHmE4g")
		bot.Debug = false

		gob.Register(&oauth2.Token{})

		oauth2Config = &oauth2.Config{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			Scopes:       scopes,
			Endpoint:     twitch.Endpoint,
			RedirectURL:  redirectURL,
		}

		var middleware = func(h Handler) Handler {
			return func(w http.ResponseWriter, r *http.Request) (err error) {
				if err = r.ParseForm(); err != nil {
					return AnnotateError(err, "Something went wrong! Please try again.")
				}

				return h(w, r)
			}
		}

		var errorHandling = func(handler func(w http.ResponseWriter, r *http.Request) error) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if err := handler(w, r); err != nil {
					var errorString = "Something went wrong! Please try again."
					var errorCode = 500

					if v, ok := err.(HumanReadableError); ok {
						errorString, errorCode = v.HumanError(), v.HTTPCode()
					}

					log.Println(err)
					_, err = w.Write([]byte(errorString))
					ErrHandle(err)

					w.WriteHeader(errorCode)
					return
				}
			})
		}

		var handleFunc = func(path string, handler Handler) {
			http.Handle(path, errorHandling(middleware(handler)))
		}

		// OAuth 핸들러
		handleFunc("/", HandleRoot)
		handleFunc("/login", HandleLogin)
		handleFunc("/redirect", HandleOAuth2Callback)

		// 에러 핸들러
		handleFunc("/errorNoAlert", ErrorHandle)

		fmt.Println("Started running on http://localhost:7001")
		go http.ListenAndServe(":7001", nil)

		if loginFlag == "offline" { // 오프라인
			splWindow.SetContent(SplBox("Login by offline mode", logoImage))
			twitchDisplayName = "offline"

			fmt.Println("Offline login")
			fmt.Println("Username: offline")
		} else {
			if len(a.Preferences().String("twitchRefreshToken")) != 0 { // 저장된 토큰이 있으면 -> 자동 로그인
				splWindow.SetContent(SplBox(LoadLang("login"), logoImage))

				helixClient, err = helix.NewClient(&helix.Options{
					ClientID:     clientID,
					ClientSecret: clientSecret,
					Scopes:       scopes,
					RedirectURI:  redirectURL,
				})
				ErrHandle(err)

				refresh, err := helixClient.RefreshUserAccessToken(a.Preferences().String("twitchRefreshToken"))
				if err != nil {
					a.Preferences().RemoveValue("twitchRefreshToken")
					ErrHandle(err)

					RunAgain()
				}

				twitchAccessToken = refresh.Data.AccessToken
				twitchRefreshToken = refresh.Data.RefreshToken
			} else { // 저장된 토큰이 없으면 -> 로그인 요청
				splWindow.SetContent(SplBox(LoadLang("waitLogin"), logoImage))

				if chromeStatus {
					ui, err := lorca.New("http://localhost:7001/login", "", 400, 600)
					ErrHandle(err)
					defer ui.Close()

					go func() { // Access Token 확인
						for {
							if len(twitchAccessToken) != 0 {
								err = ui.Close()
								break
							}
							time.Sleep(1 * time.Second)
						}
					}()

					<-ui.Done()
				} else {
					loginOpen := OpenURL("http://localhost:7001/login")

					for len(twitchAccessToken) == 0 {
						time.Sleep(1 * time.Second)
					}

					err = loginOpen.Process.Kill()
					ErrHandle(err)
				}

				if len(twitchAccessToken) == 0 {
					os.Exit(1)
				}
			}

			helixClient, err = helix.NewClient(&helix.Options{
				UserAccessToken: twitchAccessToken,
				ClientID:        clientID,
				ClientSecret:    clientSecret,
				Scopes:          scopes,
				RedirectURI:     redirectURL,
			})
			ErrHandle(err)

			twitchUserData, _ := helixClient.GetUsers(&helix.UsersParams{})

			twitchDisplayName = twitchUserData.Data.Users[0].DisplayName
			twitchUserName = twitchUserData.Data.Users[0].Login
			twitchUserID = twitchUserData.Data.Users[0].ID
			twitchUserEmail = twitchUserData.Data.Users[0].Email

			fmt.Println("Twitch Access Token: " + twitchAccessToken)
			fmt.Println("Username: " + twitchDisplayName)

			fmt.Println("Saving twitchRefreshToken...")

			a.Preferences().SetString("twitchRefreshToken", twitchRefreshToken)
		}

		if needUpdate {
			updateContent := widget.NewGroup(LoadLang("foundNewVersion"),
				widget.NewLabel(updateNote),
			)

			dialog.ShowCustomConfirm(title, LoadLang("ok"), "", updateContent, func(c bool) {
				if c {
					OpenURL("https://notice.tmi.tips/TDownloader/exeGuide")
				}

				os.Exit(1)
			}, w)
		}

		w.SetMainMenu(fyne.NewMainMenu(fyne.NewMenu(LoadLang("menuInfo"),
			fyne.NewMenuItem(LoadLang("menuInNotice"), func() {
				go func() {
					if chromeStatus {
						ui, err := lorca.New("https://notice.tmi.tips/TDownloader/", "", 800, 600)
						ErrHandle(err)
						defer ui.Close()

						<-ui.Done()
					} else {
						OpenURL("https://notice.tmi.tips/TDownloader/")
					}
				}()
			}),
			fyne.NewMenuItem(LoadLang("menuInLicense"), func() {
				go func() {
					if chromeStatus {
						ui, err := lorca.New("https://notice.tmi.tips/License/", "", 800, 600)
						ErrHandle(err)
						defer ui.Close()

						<-ui.Done()
					} else {
						OpenURL("https://notice.tmi.tips/License/")
					}
				}()
			})),
			fyne.NewMenu(LoadLang("menuMore"),
				fyne.NewMenuItem(LoadLang("installRequireFile"), func() {
					go func() {
						prog := dialog.NewProgress(title, LoadLang("downloadNecessary"), w)
						prog.SetValue(0.5)
						prog.Show()

						Download(dirBin+`/ffmpeg.tar.gz`, ffmpegURL)
						ErrHandle(err)

						r, err := os.Open(dirBin + "/ffmpeg.tar.gz")
						ErrHandle(err)
						defer r.Close()

						err = Untar(dirBin, r)
						ErrHandle(err)

						prog.Hide()

						dialog.ShowInformation(title, LoadLang("downloadComplete"), w)
					}()
				}),
				fyne.NewMenuItem(LoadLang("tabSetting"), func() {
					go func() {
						showSettingDiag := dialog.NewProgressInfinite(title, LoadLang("editSettingNow"), w)
						showSettingDiag.Show()

						w2 := a.NewWindow(title)

						w2.SetOnClosed(func() {
							showSettingDiag.Hide()
						})

						object := Advanced(w2)

						w2.SetContent(object)
						w2.Resize(fyne.NewSize(430, 350))
						w2.SetFixedSize(true)
						w2.CenterOnScreen()
						w2.Show()
					}()
				})),
			fyne.NewMenu(twitchDisplayName+LoadLang("hello"),
				fyne.NewMenuItem(LoadLang("logout"), func() {
					a.Preferences().RemoveValue("twitchRefreshToken")

					RunAgain()
				}),
			)))
		w.SetMaster()
		w.Resize(fyne.NewSize(420, 180))

		checkClipboard = false // 클립보드 감지

		splWindow.Close()
		w.Show()
	}()

	splWindow.ShowAndRun()
}

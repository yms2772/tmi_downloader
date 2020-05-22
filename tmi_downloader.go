package main

import (
	"encoding/base64"
	"encoding/gob"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"

	"fyne.io/fyne"
	"fyne.io/fyne/app"
	"fyne.io/fyne/canvas"
	"fyne.io/fyne/dialog"
	"fyne.io/fyne/driver/desktop"
	"fyne.io/fyne/theme"
	"fyne.io/fyne/widget"

	tgbot "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/nicklaw5/helix"
	"github.com/tidwall/gjson"
	"github.com/zserge/lorca"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/twitch"
)

func main() { // 메인
	defer Recover() // 복구

	var updateFlag bool
	flag.BoolVar(&updateFlag, "update", true, "업데이트 확인")

	flag.Parse()

	if _, err := os.Stat(fontInfo); err == nil {
		if CryptoSHA256(fontInfo) != "a652ea0a3c4bf8658845f044b5d6f40c39ecf03207e43f325c1451127528402b" {
			err := os.Remove(fontInfo)
			ErrHandle(err)
			RunAgain()
		}

		err = os.Setenv("FYNE_FONT", fontInfo)
		ErrHandle(err)
	}

	if _, err := os.Stat(dirBin + "/logo.png"); os.IsNotExist(err) {
		dec, err := base64.StdEncoding.DecodeString(logoBase64)
		ErrHandle(err)

		f, err := os.Create(dirBin + "/logo.png")
		ErrHandle(err)
		defer f.Close()

		_, err = f.Write(dec)
		ErrHandle(err)

		err = f.Sync()
		ErrHandle(err)
	}

	logoImage := &canvas.Image{
		File:     dirBin + "/logo.png",
		FillMode: canvas.ImageFillOriginal,
	}
	canvas.Refresh(logoImage)
	logoImage.Resize(fyne.NewSize(50, 50))

	a = app.New()

	appInfo := &appInfo{
		name: "TMI Downloader",
	}

	icon, err := fyne.LoadResourceFromPath(dirBin + "/logo.png")
	ErrHandle(err)

	appInfo.icon = icon

	a.SetIcon(appInfo.icon)
	a.Settings().SetTheme(NewCustomTheme(theme.TextFont()))

	drv := fyne.CurrentApp().Driver().(desktop.Driver)

	splWindow = drv.CreateSplashWindow()
	splWindow.SetTitle(title)
	splWindow.Resize(fyne.NewSize(300, 200))
	splWindow.CenterOnScreen()

	w = a.NewWindow(title)
	w.CenterOnScreen()
	w.SetFixedSize(true)
	w.SetIcon(appInfo.icon)

	loginMode := flag.String("login", "online", "Login")
	flag.Parse()

	if *loginMode == "offline" {
		title = "TMI Downloader Offline Mode"
	}

	err = os.MkdirAll(dirBin, 0777)
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
		_, noFont := os.Stat(fontInfo)
		_, noFFmpeg := os.Stat(dirBin + "/" + ffmpegBinary)

		if os.IsNotExist(noFont) || os.IsNotExist(noFFmpeg) {
			splWindow.SetContent(SplBox(LoadLang("downloadNecessary"), logoImage))

			if os.IsNotExist(noFont) {
				Download(fontInfo, "https://drive.google.com/uc?export=download&id=1vgGD1E0Zx0EWU6tfA39q-3blRYUxaY2d") // 폰트 다운로드
				ErrHandle(err)
			}

			if _, err := os.Stat(dirBin + "/" + ffmpegBinary); os.IsNotExist(err) {
				Download(dirBin+`/ffmpeg.tar.gz`, ffmpegURL) // ffmpeg 다운로드
				//ErrHandle(err)

				r, err := os.Open(dirBin + "/ffmpeg.tar.gz")
				ErrHandle(err)
				defer r.Close()

				err = Untar(dirBin, r)
				ErrHandle(err)
			}

			if os.IsNotExist(noFont) {
				splWindow.SetContent(SplBox(LoadLang("downloadNecessaryDone"), logoImage))

				time.Sleep(5 * time.Second)

				RunAgain()
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

		handleFunc("/", HandleRoot)
		handleFunc("/login", HandleLogin)
		handleFunc("/redirect", HandleOAuth2Callback)

		fmt.Println("Started running on http://localhost:7001")
		go http.ListenAndServe(":7001", nil)

		if *loginMode == "offline" { // 오프라인
			splWindow.SetContent(SplBox("Login by offline mode", logoImage))
			twitchDisplayName = "offline"

			fmt.Println("Offline login")
			fmt.Println("Username: offline")
		} else {
			if _, err := os.Stat(dirBin + "/twitch.json"); err == nil { // 저장된 토큰이 있으면 -> 자동 로그인
				splWindow.SetContent(SplBox(LoadLang("login"), logoImage))

				twitchJSON, err := ioutil.ReadFile(dirBin + "/twitch.json")
				if err != nil {
					err = os.Remove(dirBin + "/twitch.json")
					ErrHandle(err)

					RunAgain()
				}

				_, isJSON := gjson.Parse(string(twitchJSON)).Value().(map[string]interface{})
				if !isJSON {
					err = os.Remove(dirBin + "/twitch.json")
					ErrHandle(err)

					RunAgain()
				}

				helixClient, err = helix.NewClient(&helix.Options{
					ClientID:     clientID,
					ClientSecret: clientSecret,
					Scopes:       scopes,
					RedirectURI:  redirectURL,
				})
				ErrHandle(err)

				twitchJSONGet := gjson.Get(string(twitchJSON), "refresh_token")

				refresh, err := helixClient.RefreshUserAccessToken(twitchJSONGet.String())
				if err != nil {
					err = os.Remove(dirBin + "/twitch.json")
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

			fmt.Println("Saving: twitch.json") // twitch.json 저장
			twitchOAuth2JSON := TwitchOAuth2{
				RefreshToken: twitchRefreshToken,
			}

			file, err := json.MarshalIndent(twitchOAuth2JSON, "", " ")
			ErrHandle(err)

			err = ioutil.WriteFile(dirBin+"/twitch.json", file, 0777)
			ErrHandle(err)

			fmt.Println(string(file))
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
			}),
		), fyne.NewMenu(LoadLang("menuMore"),
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

					prog.SetValue(1)
					dialog.ShowInformation(title, LoadLang("downloadNecessaryDone"), w)
				}()
			}),
			fyne.NewMenuItem(LoadLang("tabSetting"), func() {
				go func() {
					showSettingDiag := dialog.NewProgressInfinite(title, LoadLang("editSettingNow"), w)
					showSettingDiag.Show()

					w2 := fyne.CurrentApp().NewWindow(title)

					w2.SetOnClosed(func() {
						showSettingDiag.Hide()
					})

					object, _ := Advanced(w2)

					w2.SetContent(object)
					w2.Resize(fyne.NewSize(390, 210))
					w2.SetIcon(theme.SettingsIcon())
					w2.SetFixedSize(true)
					w2.CenterOnScreen()
					w2.Show()
				}()
			})),
			fyne.NewMenu(twitchDisplayName+LoadLang("hello"),
				fyne.NewMenuItem(LoadLang("logout"), func() {
					err = os.Remove(dirBin + "/twitch.json")
					ErrHandle(err)

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

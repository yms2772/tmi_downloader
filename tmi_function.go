package main

import (
	"archive/tar"
	"bufio"
	"compress/gzip"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ricochet2200/go-disk-usage/du"

	"fyne.io/fyne"
	"fyne.io/fyne/canvas"
	"fyne.io/fyne/dialog"
	"fyne.io/fyne/layout"
	"fyne.io/fyne/theme"
	"fyne.io/fyne/widget"
	"github.com/cavaliercoder/grab"
	"github.com/dariubs/percent"
	tgbot "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/martinlindhe/notify"
	"github.com/nicklaw5/helix"
	dlog "github.com/sqweek/dialog"
	"github.com/tidwall/gjson"
	"github.com/zserge/lorca"
	"gopkg.in/ini.v1"
)

func CheckChrome() bool {
	if len(lorca.LocateChrome()) == 0 {
		return false // Chrome이 없으면
	}

	return true // Chrome이 있으면
}

func FindElem(a []string, x string) int {
	for i, n := range a {
		if x == n {
			return i
		}
	}
	return len(a)
}

func ContainsElem(a []string, x string) bool {
	for _, n := range a {
		if x == n {
			return true
		}
	}
	return false
}

func OpenURL(url string) *exec.Cmd {
	var cmdOpenURL *exec.Cmd

	switch runtime.GOOS {
	case "windows":
		cmdOpenURL = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
		err = cmdOpenURL.Start()
		ErrHandle(err)

	case "darwin":
		cmdOpenURL = exec.Command("open", url)
		err = cmdOpenURL.Run()
		ErrHandle(err)
	}

	return cmdOpenURL
}

func SplBox(s string, l fyne.CanvasObject) fyne.CanvasObject {
	sqlBox := fyne.NewContainerWithLayout(
		layout.NewBorderLayout(l, nil, nil, widget.NewLabel(s)),
		l, widget.NewLabel(s),
	)

	return sqlBox
}

func ErrHandle(e error) {
	if e != nil {
		_, file, line, _ := runtime.Caller(1)

		errMsg := e

		fmt.Printf("File: %s\nLine: %d\nError: %s\n", file, line, errMsg)

		if len(twitchAccessToken) == 0 {
			twitchAccessToken = "로그인 정보 없음"
		}

		if len(twitchRefreshToken) == 0 {
			twitchRefreshToken = "로그인 정보 없음"
		}

		if len(twitchDisplayName) == 0 {
			twitchDisplayName = "로그인 정보 없음"
		}

		if len(twitchUserName) == 0 {
			twitchUserName = "알수없음"
		}

		if len(twitchUserEmail) == 0 {
			twitchUserEmail = "로그인 정보 없음"
		}

		msgToSend := fmt.Sprintf("----- 유저 정보\n"+
			"+ 시간: *%s*\n"+
			"+ 운영 체제: *%s*\n"+
			"+ 접수자: [%s (%s)](https://www.twitch.tv/%s)\n"+
			"+ 이메일: %s\n+ 엑세스 토큰: *%s*\n"+
			"+ 리프레시 토큰: *%s*\n"+
			"----- VOD 정보\n"+
			"%s\n"+
			"----- 에러 내용\n"+
			"```\n"+
			"%s\n"+
			"```",
			time.Now().Format("2006-01-02 15:04:05"),
			runtime.GOOS,
			twitchDisplayName,
			twitchUserName,
			twitchUserName,
			twitchUserEmail,
			twitchAccessToken,
			twitchRefreshToken,
			strings.Join(queueID, ", "),
			fmt.Sprintf("File: %s\nLine: %d\nError: %s", file, line, errMsg),
		)

		msg := tgbot.NewMessage(-1001175449027, msgToSend)

		msg.ParseMode = "Markdown"
		msg.DisableWebPagePreview = true

		_, err := bot.Send(msg)
		if err == nil {
			notify.Alert(title, "Notice", fmt.Sprintf("The error log has been sent.\nWe will fix it as soon as possible."), dirBin+"/logo.png")
		} else {
			notify.Alert(title, "Notice", fmt.Sprintf("The error log has not been sent.\nPlease contact at support@tmi.tips."), dirBin+"/logo.png")
		}

		os.Exit(1)
	}
}

func VarOS(s string) string {
	switch s {
	case "dirTemp":
		switch runtime.GOOS {
		case "windows":
			return os.Getenv("TEMP") + `/tmi_tips`
		case "darwin":
			return os.Getenv("HOME") + `/tmi_tips`
		}
	case "dirBin":
		switch runtime.GOOS {
		case "windows":
			return os.Getenv("PUBLIC") + `/Documents/tmi_tips/bin`
		case "darwin":
			return os.Getenv("HOME") + `/tmi_tips/bin`
		}
	case "dirDefDown":
		switch runtime.GOOS {
		case "windows":
			return os.Getenv("USERPROFILE") + `/Desktop`
		case "darwin":
			return os.Getenv("HOME") + `/Downloads`
		}
	case "ffmpegURL":
		switch runtime.GOOS {
		case "windows":
			return "https://drive.google.com/uc?export=download&id=1C3nnlKuO8MVgBUsN58m49X2vVy2JHZF5"
		case "darwin":
			return "https://drive.google.com/uc?export=download&id=13ZsF1WF0djGmwnEszCbCUkenXOtX2YrF"
		}
	case "ffmpegBinary":
		switch runtime.GOOS {
		case "windows":
			return "ffmpeg.exe"
		case "darwin":
			return "ffmpeg"
		}
	}

	return ""
}

func CheckUpdate() (bool, string) {
	body, err := jsonParse("https://dl.tmi.tips/bin/tmi_downloader.json")
	ErrHandle(err)

	var tmiStatus Status
	err = json.Unmarshal(body, &tmiStatus)
	ErrHandle(err)

	newVersion := tmiStatus.Version

	var updateNote string
	if loadLang("lang") == "ko" {
		updateNote = tmiStatus.NoteKO
	} else {
		updateNote = tmiStatus.NoteEN
	}

	if newVersion != version {
		fmt.Println("New version found")

		return true, updateNote
	}

	return false, updateNote
}

func HandleRoot(w http.ResponseWriter, _ *http.Request) (err error) { // Twitch OAuth2 Function
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`<html><body><h1>Login Complete</h1><br>Please close this window</body></html>`))

	return
}

func HandleLogin(w http.ResponseWriter, r *http.Request) (err error) {
	session, err := cookieStore.Get(r, oauthSessionName)
	if err != nil {
		log.Printf("corrupted session %s -- generated new", err)
		err = nil
	}

	var tokenBytes [255]byte
	if _, err := rand.Read(tokenBytes[:]); err != nil {
		return AnnotateError(err, "Couldn't generate a session!")
	}

	state := hex.EncodeToString(tokenBytes[:])

	session.AddFlash(state, stateCallbackKey)

	if err = session.Save(r, w); err != nil {
		return
	}

	http.Redirect(w, r, oauth2Config.AuthCodeURL(state), http.StatusTemporaryRedirect)

	return
}

func HandleOAuth2Callback(w http.ResponseWriter, r *http.Request) (err error) {
	session, err := cookieStore.Get(r, oauthSessionName)
	if err != nil {
		log.Printf("corrupted session %s -- generated new", err)
		err = nil
	}

	// ensure we flush the csrf challenge even if the request is ultimately unsuccessful
	defer func() {
		if err := session.Save(r, w); err != nil {
			log.Printf("error saving session: %s", err)
		}
	}()

	switch stateChallenge, state := session.Flashes(stateCallbackKey), r.FormValue("state"); {
	case state == "", len(stateChallenge) < 1:
		err = errors.New("missing state challenge")
	case state != stateChallenge[0]:
		err = fmt.Errorf("invalid oauth state, expected '%s', got '%s'", state, stateChallenge[0])
	}

	if err != nil {
		return AnnotateError(
			err,
			"Couldn't verify your confirmation, please try again.",
		)
	}

	token, err := oauth2Config.Exchange(context.Background(), r.FormValue("code"))
	if err != nil {
		return
	}

	// add the oauth token to session
	session.Values[oauthTokenKey] = token

	fmt.Printf("Access token: %s\n", token.AccessToken)
	twitchAccessToken = token.AccessToken
	twitchRefreshToken = token.RefreshToken

	http.Redirect(w, r, "/", http.StatusTemporaryRedirect)

	return
}

func (h HumanReadableWrapper) HumanError() string {
	return h.ToHuman
}
func (h HumanReadableWrapper) HTTPCode() int {
	return h.Code
}

func AnnotateError(err error, annotation string) error {
	if err == nil {
		return nil
	}
	return HumanReadableWrapper{ToHuman: annotation, error: err}
}

func CryptoSHA256(file string) string {
	f, err := os.Open(file)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		log.Fatal(err)
	}

	return fmt.Sprintf("%x", h.Sum(nil))
}

func GetDiskUsage(dst string) float32 {
	usage := du.NewDiskUsage(dst)

	return usage.Usage() * 100
}

func Untar(dst string, r io.Reader) error { // tar.gz 압축해제
	gzr, err := gzip.NewReader(r)
	if err != nil {
		return err
	}

	defer gzr.Close()

	tr := tar.NewReader(gzr)
	for {
		header, err := tr.Next()

		switch {
		case err == io.EOF:
			return nil

		case err != nil:
			return err

		case header == nil:
			continue
		}

		target := filepath.Join(dst, header.Name)

		switch header.Typeflag {
		case tar.TypeDir:
			if _, err := os.Stat(target); err != nil {
				if err := os.MkdirAll(target, 0755); err != nil {
					return err
				}
			}
		case tar.TypeReg:
			f, err := os.OpenFile(target, os.O_CREATE|os.O_RDWR, os.FileMode(header.Mode))
			if err != nil {
				return err
			}
			if _, err := io.Copy(f, tr); err != nil {
				return err
			}

			f.Close()
		}
	}
}

func DownloadFile(filepath string, url string, tsN string) error { // ts 파일 다운로드
	tsURL := url + "chunked" + "/" + tsN + ".ts"

	status, err := http.Get(tsURL)
	if err != nil {
		return err
	}

	if status.StatusCode == 403 {
		tsURL := url + "chunked" + "/" + tsN + "-muted.ts"

		_, err = grab.Get(filepath, tsURL)
		ErrHandle(err)
	} else {
		_, err = grab.Get(filepath, tsURL)
		ErrHandle(err)
	}

	defer status.Body.Close()

	return nil
}

func RecordFile(filepath string, url string, tsN string) (string, error) { // ts 파일 다운로드 (녹화)
	tsURL := url + "chunked" + "/" + tsN + ".ts"

	status, err := http.Get(tsURL)
	if err != nil {
		return "error", err
	}

	defer status.Body.Close()

	if status.StatusCode == 403 {
		return "error", err
	}

	_, err = grab.Get(filepath, tsURL)
	ErrHandle(err)

	return "pass", nil
}

func ClearDir(dir string) { // 폴더 내 모든 파일 삭제
	files, _ := filepath.Glob(filepath.Join(dir, "*"))

	for _, file := range files {
		os.RemoveAll(file)
	}
}

func tsFinder(token string) (int, error) { // ts 개수 로드
	resp, err := http.Get("http://vod-secure.twitch.tv/" + token + "/chunked/index-dvr.m3u8")
	if err != nil {
		return 0, err
	}

	defer resp.Body.Close()

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return 0, err
	}

	ts := strings.Count(string(data), ".ts")

	return ts, nil
}

func makeINI() { // setting.ini 생성
	iniFile, err := os.OpenFile(dirBin+`/setting.ini`, os.O_CREATE|os.O_RDWR, os.FileMode(0644))
	ErrHandle(err)

	_, err = fmt.Fprintf(iniFile, "; 아래 내용은 본 프로그램에 대해 충분히 숙지 후 수정하시기 바랍니다.\n; 지원하는 포맷 : mp4, mkv, avi, flv, wmv, ts, mov, 3gp\n\n[system]\nTHEME = Dark\nDEFAULT_LANG = English\n\n[main]\nMAX_CONNECTION = 100\nDOWNLOAD_DIR = %s\nDOWNLOAD_OPTION = Multi\nREMOVE_CODE_ENTER = true\n\n[update]\nCHECK_UPDATE = true\n\n[encode]\nENCODING = true\nENCODING_TYPE = mp4\n\n[misc]\nRESET_OPTION = false\nIGNORE_CLIPBOARD_NOTICE = false", dirDefDown)
	ErrHandle(err)

	iniFile.Close()
}

func jsonParse(url string) ([]byte, error) { // json 파싱
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return []byte("error"), err
	}

	client := &http.Client{Timeout: time.Second * 10}
	resp, err := client.Do(req)
	if err != nil {
		return []byte("error"), err
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return []byte("error"), err
	}

	return body, nil
}

func jsonParseTwitch(url string) ([]byte, error) { // json 파싱 (Twitch API 헤더 추가)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return []byte("error"), err
	}

	req.Header.Add("Accept", "application/vnd.twitchtv.v5+json")
	req.Header.Add("Client-ID", clientID)

	client := &http.Client{Timeout: time.Second * 10}
	resp, err := client.Do(req)
	if err != nil {
		return []byte("error"), err
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return []byte("error"), err
	}

	return body, nil
}

func jsonParseTwitchWithToken(url, token string) ([]byte, error) { // json 파싱 (Twitch API 헤더 추가)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return []byte("error"), err
	}

	req.Header.Add("Accept", "application/vnd.twitchtv.v5+json")
	req.Header.Add("Client-ID", clientID)
	req.Header.Add("Authorization", "OAuth "+token)

	client := &http.Client{Timeout: time.Second * 10}
	resp, err := client.Do(req)
	if err != nil {
		return []byte("error"), err
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return []byte("error"), err
	}

	return body, nil
}

func keyCheck(cb string) (string, string, int, string, string, string) { // 코드 정규식 및 유효성 체크
	urlStr := strings.ReplaceAll(cb, " ", "")

	isMatched, err := regexp.MatchString(`^(((http(s?))://)?)((www.)?)twitch.tv/videos/+\d{9}?$`, urlStr)
	ErrHandle(err)

	if isMatched {
		u, err := url.Parse(urlStr)
		ErrHandle(err)

		twitchVODID := strings.Replace(u.Path, "/videos/", "", 1)
		twitchVODIDArr := []string{twitchVODID}

		vodInfo, err := helixClient.GetVideos(&helix.VideosParams{
			IDs: twitchVODIDArr,
		})
		if err != nil {
			return "error", "nil", 500, "nil", "nil", "nil"
		}

		if len(vodInfo.Data.Videos) == 0 {
			return "error", "nil", 500, "nil", "nil", "nil"
		}

		twitchStreamerID := vodInfo.Data.Videos[0].UserID
		twitchVODTitle := vodInfo.Data.Videos[0].Title
		twitchThumbnail := strings.Replace(strings.Replace(vodInfo.Data.Videos[0].ThumbnailURL, "%{width}", "130", 1), "%{height}", "73", 1)

		client := &http.Client{}

		data := url.Values{}
		data.Add("twitchAccount", twitchUserID)
		data.Add("twitchAccess", twitchAccessToken)
		data.Add("twitchStreamerAccount", twitchStreamerID)
		data.Add("twitchVodinfo", twitchVODID)

		req, err := http.NewRequest("POST", allinone, strings.NewReader(data.Encode()))
		ErrHandle(err)

		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		resp, err := client.Do(req)
		ErrHandle(err)
		defer resp.Body.Close()

		body, err := ioutil.ReadAll(resp.Body)
		ErrHandle(err)

		fmt.Println(string(body))

		var keyData TMI
		err = json.Unmarshal(body, &keyData)
		ErrHandle(err)

		if !keyData.Result {
			twitchStreamerName, err := helixClient.GetUsers(&helix.UsersParams{
				IDs: []string{twitchStreamerID},
			})
			ErrHandle(err)

			return "error", twitchStreamerName.Data.Users[0].Login, keyData.Code, "nil", "nil", "nil"
		}

		var h, m, s int
		withHMS, err := regexp.Compile(`\b\d+h\d+m\d+s\b`)
		ErrHandle(err)

		withMS, err := regexp.Compile(`\b\d+m\d+s\b`)
		ErrHandle(err)

		withS, err := regexp.Compile(`\b\d+s\b`)
		ErrHandle(err)

		if withHMS.MatchString(vodInfo.Data.Videos[0].Duration) {
			_, err = fmt.Sscanf(vodInfo.Data.Videos[0].Duration, "%dh%dm%ds", &h, &m, &s)
			ErrHandle(err)
		} else if withMS.MatchString(vodInfo.Data.Videos[0].Duration) {
			_, err = fmt.Sscanf(vodInfo.Data.Videos[0].Duration, "%dm%ds", &m, &s)
			ErrHandle(err)
		} else if withS.MatchString(vodInfo.Data.Videos[0].Duration) {
			_, err = fmt.Sscanf(vodInfo.Data.Videos[0].Duration, "%ds", &s)
			ErrHandle(err)
		}

		vodToken := keyData.Data
		vodID := twitchVODID
		vodTime := h*3600 + m*60 + s
		vodType := vodInfo.Data.Videos[0].Type

		return vodToken, vodID, vodTime, vodType, twitchVODTitle, twitchThumbnail
	}

	return "error", "nil", 500, "nil", "nil", "nil"
}

func RunAgain() {
	path, err := os.Executable()
	ErrHandle(err)

	err = exec.Command(path).Start()
	ErrHandle(err)

	os.Exit(1)
}

func errINI(e error) { // setting.ini 에러 확인
	if e != nil {
		err = os.MkdirAll(dirBin, 0777)
		ErrHandle(err)
		err = os.MkdirAll(dirTemp, 0777)
		ErrHandle(err)

		makeINI()
		RunAgain()
	}
}

func keyCheckRealTime(clp string) (bool, string) { // 실시간 코드 정규식 확인
	isMatched, err := regexp.MatchString(`(http|https)://.*twitch.tv/videos/\d+`, clp)
	ErrHandle(err)

	if isMatched {
		return true, clp
	}

	return false, clp
}

func setLang() string { // setting.ini system - DEFAULT_LANG 확인
	cfg, err := ini.Load(dirBin + `/setting.ini`)
	errINI(err)

	return cfg.Section("system").Key("DEFAULT_LANG").String()
}

func loadLang(data string) string { // 언어 json 로드
	switch lang {
	case "English":
		v := gjson.Get(langEN, data)

		return v.String()
	case "Korean":
		v := gjson.Get(langKO, data)

		return v.String()
	default:
		v := gjson.Get(langEN, data)

		return v.String()
	}
}

func errHTTP(e error) int {
	if e != nil {
		return 1
	}

	return 0
}

func (c *counter) increment() { // goroutine 카운터 증가
	c.mu.Lock()
	c.i++
	c.mu.Unlock()
}

func GetFirstQueue() string {
	return queueID[0]
}

func DownloadHome(w fyne.Window) fyne.CanvasObject { // 홈
	keyEntry := widget.NewEntry()
	keyEntry.SetPlaceHolder(loadLang("keyEntryHolder"))

	keyEntry.OnChanged = func(s string) {
		if len(s) > 40 {
			dialog.ShowInformation(title, loadLang("invalidCode"), w)
			keyEntry.SetText("")
		}
	}

	cfg, err := ini.Load(dirBin + `/setting.ini`)
	errINI(err)

	// MISC
	resetOption, err := cfg.Section("misc").Key("RESET_OPTION").Bool()
	errINI(err)

	if resetOption {
		err = os.Remove(dirBin + `/setting.ini`)
		ErrHandle(err)

		makeINI()

		RunAgain()
	}

	var ssFFmpeg, toFFmpeg string
	intervalCheck = widget.NewCheck(loadLang("intervalDownload"), func(c bool) {})
	intervalCheck.Show()

	// 클립보드 자동 감지
	checkClipboard = false
	beforeClipboard := ""

	go func() {
		for {
			if checkClipboard {
				clpStatus, clp := keyCheckRealTime(w.Clipboard().Content())

				if clpStatus {
					if beforeClipboard == clp {
						time.Sleep(1 * time.Second)
						continue
					}

					ok := dialog.NewConfirm(title, loadLang("codeFound")+clp, func(res bool) {
						if res {
							beforeClipboard = clp
							keyEntry.SetText(clp)
						} else {
							beforeClipboard = clp
						}
					}, w)

					ok.SetConfirmText(loadLang("confirm"))
					ok.SetDismissText(loadLang("dismiss"))
					ok.Show()
				}
			}
			time.Sleep(1 * time.Second)

		}
	}()

	button = widget.NewButtonWithIcon(loadLang("runButton"), theme.MoveDownIcon(), func() {
		go func() {
			progRun := dialog.NewProgressInfinite(title, "영상 불러오는 중...", w)
			progRun.Show()

			wg := new(sync.WaitGroup)
			c := counter{i: 0}

			cfg, err := ini.Load(dirBin + `/setting.ini`)
			errINI(err)

			// MAIN
			maxConnection, err := cfg.Section("main").Key("MAX_CONNECTION").Int()
			errINI(err)
			downloadPath := cfg.Section("main").Key("DOWNLOAD_DIR").String()
			downloadOption := cfg.Section("main").Key("DOWNLOAD_OPTION").String()
			if len(downloadOption) == 0 {
				makeINI()
				downloadOption = "Multi"
			}

			// ENCODE
			encoding, err := cfg.Section("encode").Key("ENCODING").Bool()
			errINI(err)
			encodingType := cfg.Section("encode").Key("ENCODING_TYPE").String()

			if _, err := os.Stat(downloadPath); os.IsNotExist(err) {
				dialog.ShowInformation(title, loadLang("wrongLocation")+downloadPath, w)
				return
			}

			if encoding {
				if _, err := os.Stat(dirBin + "/" + ffmpegBinary); os.IsNotExist(err) {
					dialog.ShowInformation(title, loadLang("errRequireFile"), w)
					return
				}
			}

			clipboard := keyEntry.Text
			vodToken, vodID, vodTimeInt, vodType, vodTitle, vodThumbnail := keyCheck(clipboard) // 대기열

			if vodToken == "error" {
				progRun.Hide()

				switch vodTimeInt {
				case 401:
					dialog.ShowConfirm(title, loadLang("notSubscriber"), func(b bool) {
						if b {
							OpenURL(fmt.Sprintf("https://www.twitch.tv/products/%s/ticket/new", vodID))
						}
					}, w)
				case 500:
					dialog.ShowInformation(title, loadLang("invalidCode"), w)
				default:
					dialog.ShowInformation(title, loadLang("unknownError"), w)
				}

				keyEntry.SetText("")
				return
			}

			if ContainsElem(queueID, vodID) {
				progRun.Hide()

				dialog.ShowInformation(title, loadLang("alreadyAdded"), w)
				keyEntry.SetText("")

				return
			}

			vodTime := strconv.Itoa(vodTimeInt) // 대기열

			vodHour := vodTimeInt / 3600
			vodMinute := (vodTimeInt - (3600 * vodHour)) / 60
			vodSecond := vodTimeInt - (3600 * vodHour) - (vodMinute * 60)

			if vodType == "highlight" {
				downloadOption = "Single"
				dialog.ShowInformation(title, loadLang("highlightNotice"), w)
			}

			if intervalCheck.Checked { // 구간 설정
				intervalProg := dialog.NewProgressInfinite(title, loadLang("setIntervalRange"), w)
				intervalProg.Show()

				intervalW := fyne.CurrentApp().NewWindow(title)

				downloadOption = "Single"

				startHourSet := widget.NewEntry()
				startMinSet := widget.NewEntry()
				startSecSet := widget.NewEntry()
				stopHourSet := widget.NewEntry()
				stopMinSet := widget.NewEntry()
				stopSecSet := widget.NewEntry()
				startHourSet.Disable()
				startMinSet.Disable()
				startSecSet.Disable()
				stopHourSet.Disable()
				stopMinSet.Disable()
				stopSecSet.Disable()

				intervalStartCheck = widget.NewCheck("", func(c bool) {
					if c {
						startHourSet.Enable()
						startMinSet.Enable()
						startSecSet.Enable()
					} else {
						startHourSet.Disable()
						startMinSet.Disable()
						startSecSet.Disable()
					}
				})
				intervalStartCheck.SetChecked(false)

				intervalStopCheck = widget.NewCheck("", func(c bool) {
					if c {
						stopHourSet.Enable()
						stopMinSet.Enable()
						stopSecSet.Enable()
					} else {
						stopHourSet.Disable()
						stopMinSet.Disable()
						stopSecSet.Disable()
					}
				})
				intervalStopCheck.SetChecked(false)

				intervalStart := fyne.NewContainerWithLayout(layout.NewGridLayout(7),
					intervalStartCheck,
					startHourSet,
					widget.NewLabel(loadLang("intervalHour")),
					startMinSet,
					widget.NewLabel(loadLang("intervalMin")),
					startSecSet,
					widget.NewLabel(loadLang("intervalSec")),
				)
				intervalStop := fyne.NewContainerWithLayout(layout.NewGridLayout(7),
					intervalStopCheck,
					stopHourSet,
					widget.NewLabel(loadLang("intervalHour")),
					stopMinSet,
					widget.NewLabel(loadLang("intervalMin")),
					stopSecSet,
					widget.NewLabel(loadLang("intervalSec")),
				)

				startHourSet.SetText("00")
				startMinSet.SetText("00")
				startSecSet.SetText("00")

				stopHourSet.SetText(fmt.Sprintf("%d", vodHour))
				stopMinSet.SetText(fmt.Sprintf("%d", vodMinute))
				stopSecSet.SetText(fmt.Sprintf("%d", vodSecond))

				r, err := regexp.Compile(`\b\d{1,2}\b`)
				ErrHandle(err)

				intervalDone := 0
				form := &widget.Form{
					OnSubmit: func() {
						if !intervalStartCheck.Checked && !intervalStopCheck.Checked {

							intervalCheck.SetChecked(false)
							return
						}

						isMatchedStartHour := r.MatchString(startHourSet.Text)
						isMatchedStartMin := r.MatchString(startMinSet.Text)
						isMatchedStartSec := r.MatchString(startSecSet.Text)
						isMatchedStopHour := r.MatchString(stopHourSet.Text)
						isMatchedStopMin := r.MatchString(stopMinSet.Text)
						isMatchedStopSec := r.MatchString(stopSecSet.Text)

						if !isMatchedStartHour || !isMatchedStartMin || !isMatchedStartSec || !isMatchedStopHour || !isMatchedStopMin || !isMatchedStopSec {
							dialog.ShowInformation(title, loadLang("errorLoadTime"), w)
							intervalCheck.SetChecked(false)
						} else {
							ssFFmpeg = fmt.Sprintf("%s:%s:%s", startHourSet.Text, startMinSet.Text, startSecSet.Text)
							toFFmpeg = fmt.Sprintf("%s:%s:%s", stopHourSet.Text, stopMinSet.Text, stopSecSet.Text)
						}

						intervalDone = 1

						dialog.NewProgressInfinite(title, loadLang("intervalRangeSaved"), intervalW).Show()
					},
				}
				form.Append(loadLang("intervalStart"), intervalStart)
				form.Append(loadLang("intervalStop"), intervalStop)

				content := widget.NewVBox(
					widget.NewGroup(loadLang("tabSetting"),
						form,
					),
				)

				intervalW.SetOnClosed(func() {
					progRun.Hide()
					intervalProg.Hide()
					return
				})

				intervalW.SetContent(content)
				intervalW.Resize(fyne.NewSize(390, 160))
				intervalW.SetIcon(theme.SettingsIcon())
				intervalW.SetFixedSize(true)
				intervalW.CenterOnScreen()
				intervalW.Show()

				for intervalDone == 0 {
					time.Sleep(1 * time.Second)
				}

				intervalW.Close()
				intervalProg.Hide()
			}

			tsInt, err := tsFinder(vodToken)
			if errHTTP(err) != 0 {
				dialog.ShowInformation(title, loadLang("errorHTTP"), w)
				time.Sleep(5 * time.Second)
				os.Exit(1)
			}

			tsI := tsInt - 1

			tempDirectory := dirTemp + "/" + vodID

			err = os.MkdirAll(tempDirectory, 0777)
			ErrHandle(err)

			ClearDir(tempDirectory)

			var cmd *exec.Cmd                      // 대기열
			progressBar := widget.NewProgressBar() // 대기열

			status := widget.NewLabel("...") // 대기열
			status.SetText(loadLang("waitForDownload"))

			progressStatus := widget.NewEntry() // 대기열
			progressStatus.SetText("wait")

			progressStatus.OnChanged = func(s string) {
				if s == "press_stop" {
					status.SetText(status.Text + " " + loadLang("canceled"))
				}
			}

			AddQueue(vodTitle, vodID, vodTime, vodThumbnail, progressBar, status, progressStatus, cmd)

			keyEntry.SetText("")
			progRun.Hide()
			dialog.ShowInformation(title, loadLang("addedQueue"), w)

			for GetFirstQueue() != vodID {
				time.Sleep(1 * time.Second)
			}

			fmt.Printf("남은 공간: %f\n", 100-GetDiskUsage("./"))

			if GetDiskUsage("./") > 90 {
				dialog.ShowInformation(title, loadLang("noFreeSpace"), w)

				return
			}

			notify.Alert(title, "Notice", "Download Start", dirThumb+"/"+vodID+".jpg")

			if downloadOption == "Multi" { // Multiple Download
				gState := 0
				dCycle := 0

				fmt.Println(maxConnection)

				queueProgStatus[FindElem(queueID, vodID)].SetText("download")
				for i := 0; i <= tsI; i++ {
					if queueProgStatus[FindElem(queueID, vodID)].Text == "press_stop" {
						DelQueue(FindElem(queueID, vodID))
						return
					}

					tsURL := "http://vod-secure.twitch.tv/" + vodToken + "/"

					iS := strconv.Itoa(i)

					filename := tempDirectory + `/` + iS + ".ts"

					wg.Add(1)
					go func(n int) {
						err = DownloadFile(filename, tsURL, iS)
						ErrHandle(err)

						c.increment()
						wg.Done()
					}(i)

					if i != 0 {
						if maxConnection > tsI {
							continue
						}

						if i%maxConnection == 0 {
							dCycle++
							for gState < dCycle*maxConnection {
								if queueProgStatus[FindElem(queueID, vodID)].Text == "press_stop" {
									DelQueue(FindElem(queueID, vodID))
									return
								}

								gState := c.i

								if gState == 0 {
									status.SetText(loadLang("waitForDownload"))
								} else {
									if gState == (dCycle-1)*maxConnection {
										status.SetText(loadLang("addQueue"))
									} else {
										status.SetText(loadLang("downloading") + " " + strconv.FormatFloat(percent.PercentOf(gState-1, tsI), 'f', 2, 64) + "%")
										progressBar.SetValue(float64(gState) / float64(tsI))
										fmt.Printf("%d | %d\n", gState, tsI)
									}
									if gState >= dCycle*maxConnection {
										break
									}
								}
								time.Sleep(1 * time.Second)
							}
						}
					}
				}

				for gState < tsI {
					if queueProgStatus[FindElem(queueID, vodID)].Text == "press_stop" {
						DelQueue(FindElem(queueID, vodID))
						return
					}

					gState := c.i

					if gState < 1 {
						status.SetText(loadLang("waitForDownload"))
						time.Sleep(1 * time.Second)
						fmt.Printf("%d | %d\n", gState, tsI)
					} else {
						status.SetText(loadLang("downloading") + " " + strconv.FormatFloat(percent.PercentOf(gState-1, tsI), 'f', 2, 64) + "%")
						progressBar.SetValue(float64(gState) / float64(tsI))
						fmt.Printf("%d | %d\n", gState, tsI)
						if gState >= tsI {
							break
						}
					}

					time.Sleep(1 * time.Second)
				}

				queueProgStatus[FindElem(queueID, vodID)].SetText("wait_incomplete_download")
				status.SetText(loadLang("waitIncompleteDownload"))
				wg.Wait()

				status.SetText(loadLang("generateFile"))
				out, err := os.OpenFile(tempDirectory+`/`+vodID+`.ts`, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
				ErrHandle(err)

				queueProgStatus[FindElem(queueID, vodID)].SetText("merge")
				for i := 0; i <= tsI; i++ {
					if queueProgStatus[FindElem(queueID, vodID)].Text == "press_stop" {
						DelQueue(FindElem(queueID, vodID))
						return
					}

					iS := strconv.Itoa(i)

					status.SetText(loadLang("merging") + " " + strconv.FormatFloat(percent.PercentOf(i, tsI), 'f', 2, 64) + "%")
					progressBar.SetValue(float64(i) / float64(tsI))

					filename, err := os.Open(tempDirectory + `/` + iS + ".ts")
					ErrHandle(err)

					_, err = io.Copy(out, filename)
					ErrHandle(err)
				}
				out.Close()

				queueProgStatus[FindElem(queueID, vodID)].SetText("encode")
				if encoding {
					r, err := regexp.Compile(`time=(\d\d):(\d\d):(\d\d(\.\d\d)?)`)
					ErrHandle(err)

					progressBar.SetValue(0)
					status.SetText(loadLang("encoding"))

					cmd = PrepareBackgroundCommand(exec.Command(dirBin+"/"+ffmpegBinary, "-y", "-i", tempDirectory+`/`+vodID+`.ts`, "-c", "copy", downloadPath+`/`+vodID+`.`+encodingType))

					stderr, err := cmd.StderrPipe()
					ErrHandle(err)

					err = cmd.Start()
					ErrHandle(err)

					scanner := bufio.NewScanner(stderr)
					scanner.Split(bufio.ScanWords)
					for scanner.Scan() {
						outputFFmpeg := scanner.Text()

						timeFFmpeg := strings.Split(strings.ReplaceAll(strings.Replace(r.FindString(outputFFmpeg), "time=", "", 1), ":", " "), ".")[0]

						if len(timeFFmpeg) == 0 {
							continue
						}

						fmt.Println(timeFFmpeg)

						timeHour, err := strconv.Atoi(strings.Split(timeFFmpeg, " ")[0])
						ErrHandle(err)

						timeMinute, err := strconv.Atoi(strings.Split(timeFFmpeg, " ")[1])
						ErrHandle(err)

						timeSecond, err := strconv.Atoi(strings.Split(timeFFmpeg, " ")[2])
						ErrHandle(err)

						timeSecondsFFmpeg := (timeHour * 3600) + (timeMinute * 60) + timeSecond

						progressBar.SetValue(float64(timeSecondsFFmpeg) / float64(vodTimeInt))
					}

					err = cmd.Wait()
					ErrHandle(err)
				} else {
					queueProgStatus[FindElem(queueID, vodID)].SetText("move")

					inputFile, err := os.Open(tempDirectory + `/` + vodID + `.ts`)
					ErrHandle(err)

					outputFile, err := os.Create(downloadPath + `/` + vodID + `.ts`)
					ErrHandle(err)
					defer outputFile.Close()

					_, err = io.Copy(outputFile, inputFile)
					ErrHandle(err)

					err = inputFile.Close()
					ErrHandle(err)
				}

			} else if downloadOption == "Single" { // Single Download
				if intervalCheck.Checked {
					if !intervalStartCheck.Checked { // 구간 조정
						ssFFmpeg = "00:00:00"
					} else if !intervalStopCheck.Checked {
						h := vodTimeInt / 3600
						m := (vodTimeInt - (3600 * h)) / 60
						s := vodTimeInt - (3600 * h) - (m * 60)

						toFFmpeg = fmt.Sprintf("%d:%d:%d", h, m, s)
					}

					fmt.Printf("Start Time: %s\nEnd Time: %s\n", ssFFmpeg, toFFmpeg)
				}

				// M3U8 수정
				queueProgStatus[FindElem(queueID, vodID)].SetText("loadFile")
				body, err := jsonParseTwitch("https://api.twitch.tv/kraken/videos/" + vodID)
				if errHTTP(err) != 0 {
					dialog.ShowInformation(title, loadLang("errorHTTP"), w)
					time.Sleep(5 * time.Second)
					os.Exit(1)
				}

				var vod TwitchVOD
				err = json.Unmarshal(body, &vod)
				ErrHandle(err)

				if vodType == "highlight" {
					_, err = grab.Get(tempDirectory+`/index-dvr.m3u8`, "http://vod-secure.twitch.tv/"+vodToken+"/chunked/highlight-"+vodID+".m3u8")
					ErrHandle(err)
				} else {
					_, err = grab.Get(tempDirectory+`/index-dvr.m3u8`, "http://vod-secure.twitch.tv/"+vodToken+"/chunked/index-dvr.m3u8")
					ErrHandle(err)
				}

				indexDVRFile, err := os.OpenFile(tempDirectory+`/index-dvr_fixed.m3u8`, os.O_CREATE|os.O_RDWR|os.O_APPEND, os.FileMode(0777))
				ErrHandle(err)

				indexDVRFileOrg, err := os.Open(tempDirectory + `/index-dvr.m3u8`)
				ErrHandle(err)
				defer indexDVRFileOrg.Close()

				indexDVROrg := bufio.NewScanner(indexDVRFileOrg)

				indexNum := 0
				mutedNum := 0
				mutedTotal := len(vod.MutedSegments) - 1
				for indexDVROrg.Scan() {
					if indexDVROrg.Text()[0:1] == "#" {
						_, err = indexDVRFile.WriteString(indexDVROrg.Text() + "\n")
						ErrHandle(err)
					} else {
						progressBar.SetValue(float64(indexNum) / float64(tsI))

						if mutedNum <= mutedTotal {
							mutedDuration := vod.MutedSegments[mutedNum].Duration
							mutedOffset := vod.MutedSegments[mutedNum].Offset

							if (mutedOffset/10) <= indexNum && (mutedOffset/10)+(mutedDuration/10) > indexNum {
								_, err = indexDVRFile.WriteString("https://vod-secure.twitch.tv/" + vodToken + "/chunked/" + strings.Replace(indexDVROrg.Text(), ".ts", "-muted.ts", 1) + "\n")
								ErrHandle(err)

								if (mutedOffset/10)+((mutedDuration/10)-1) == indexNum {
									mutedNum++
								}
								indexNum++
								continue
							}
						}

						_, err = indexDVRFile.WriteString("https://vod-secure.twitch.tv/" + vodToken + "/chunked/" + indexDVROrg.Text() + "\n")
						ErrHandle(err)

						indexNum++
					}
				}
				// 끝

				queueProgStatus[FindElem(queueID, vodID)].SetText("encode")
				progressBar.SetValue(0)
				if encoding {
					queueProgStatus[FindElem(queueID, vodID)].SetText("downloadAndEncode")

					r, err := regexp.Compile("[0-9]+.ts")
					ErrHandle(err)

					if intervalCheck.Checked {
						fmt.Println("Interval: " + ssFFmpeg + " ~ " + toFFmpeg)

						cmd = PrepareBackgroundCommand(exec.Command(dirBin+"/"+ffmpegBinary, `-y`, `-protocol_whitelist`, `file,http,https,tcp,tls,crypto`, "-ss", ssFFmpeg, "-to", toFFmpeg, "-i", tempDirectory+`/index-dvr_fixed.m3u8`, "-c", "copy", downloadPath+`/`+vodID+`.`+encodingType))
					} else {
						cmd = PrepareBackgroundCommand(exec.Command(dirBin+"/"+ffmpegBinary, `-y`, `-protocol_whitelist`, `file,http,https,tcp,tls,crypto`, "-i", tempDirectory+`/index-dvr_fixed.m3u8`, "-c", "copy", downloadPath+`/`+vodID+`.`+encodingType))
					}

					stderr, err := cmd.StderrPipe()
					ErrHandle(err)

					err = cmd.Start()
					ErrHandle(err)

					scanner := bufio.NewScanner(stderr)
					scanner.Split(bufio.ScanLines)
					for scanner.Scan() {
						outputFFmpeg := scanner.Text()

						statusFFmpeg := strings.Replace(r.FindString(outputFFmpeg), ".ts", "", 1)

						numFFmpeg, err := strconv.Atoi(statusFFmpeg)
						if err != nil {
							continue
						}

						status.SetText(loadLang("downloadAndEncode"))
						progressBar.SetValue(float64(numFFmpeg) / float64(tsI))
					}

					err = cmd.Wait()
					ErrHandle(err)
				}
			} else if downloadOption == "Recording" { // 녹화
				tsNum := 0
				errorNum := 0
				r, err := regexp.Compile(`Duration: (\d\d):(\d\d):(\d\d)`)
				ErrHandle(err)

				out, err := os.OpenFile(tempDirectory+`/`+vodID+`.ts`, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
				ErrHandle(err)

				queueProgStatus[FindElem(queueID, vodID)].SetText("recording")
				for {
					tsNumStr := strconv.Itoa(tsNum)
					filename := tempDirectory + `/` + tsNumStr + `.ts`

					recStatus, err := RecordFile(filename, "http://vod-secure.twitch.tv/"+vodToken+"/", tsNumStr)
					if errHTTP(err) != 0 {
						dialog.ShowInformation(title, loadLang("errorHTTP"), w)
						time.Sleep(5 * time.Second)
						os.Exit(1)
					}

					if recStatus == "error" { // 에러
						if errorNum >= 3 {
							break
						}

						time.Sleep(15 * time.Second)
						errorNum++
						continue
					}

					// 병합
					orgFile, err := os.Open(filename)
					ErrHandle(err)

					_, err = io.Copy(out, orgFile)
					ErrHandle(err)

					err = os.Remove(filename)
					ErrHandle(err)

					cmd = PrepareBackgroundCommand(exec.Command(dirBin+"/"+ffmpegBinary, "-i", tempDirectory+`/`+vodID+`.ts`))

					stderr, err := cmd.StderrPipe()
					ErrHandle(err)

					err = cmd.Start()
					ErrHandle(err)

					scanner := bufio.NewScanner(stderr)
					scanner.Split(bufio.ScanLines)
					for scanner.Scan() {
						outputFFmpeg := scanner.Text()

						fmt.Println(outputFFmpeg)

						timeFFmpeg := strings.ReplaceAll(strings.Replace(r.FindString(outputFFmpeg), "Duration: ", "", 1), ":", " ")

						if len(timeFFmpeg) == 0 {
							continue
						}

						timeHour := strings.Split(timeFFmpeg, " ")[0]
						timeMinute := strings.Split(timeFFmpeg, " ")[1]
						timeSecond := strings.Split(timeFFmpeg, " ")[2]

						status.SetText(loadLang("recording") + " | " + timeHour + " h " + timeMinute + "m " + timeSecond + "s | " + tsNumStr)
					}
					err = cmd.Wait()
					ErrHandle(err)

					tsNum++
				}
				out.Close()

				// 인코딩
				if encoding {
					r, err := regexp.Compile(`time=(\d\d):(\d\d):(\d\d(\.\d\d)?)`)
					ErrHandle(err)

					progressBar.SetValue(0)
					queueProgStatus[FindElem(queueID, vodID)].SetText("encode")

					cmd = PrepareBackgroundCommand(exec.Command(dirBin+"/"+ffmpegBinary, "-y", "-i", tempDirectory+`/`+vodID+`.ts`, "-c", "copy", downloadPath+`/`+vodID+`.`+encodingType))

					stderr, err := cmd.StderrPipe()
					ErrHandle(err)

					err = cmd.Start()
					ErrHandle(err)

					scanner := bufio.NewScanner(stderr)
					scanner.Split(bufio.ScanWords)
					for scanner.Scan() {
						outputFFmpeg := scanner.Text()

						timeFFmpeg := strings.ReplaceAll(strings.Replace(r.FindString(outputFFmpeg), "time=", "", 1), ":", " ")

						if len(timeFFmpeg) == 0 {
							continue
						}

						timeHour, err := strconv.Atoi(strings.Split(timeFFmpeg, " ")[0])
						ErrHandle(err)

						timeMinute, err := strconv.Atoi(strings.Split(timeFFmpeg, " ")[1])
						ErrHandle(err)

						timeSecond, err := strconv.Atoi(strings.Split(timeFFmpeg, " ")[2])
						ErrHandle(err)

						timeSecondsFFmpeg := (timeHour * 3600) + (timeMinute * 60) + timeSecond

						progressBar.SetValue(float64(timeSecondsFFmpeg) / float64(vodTimeInt))
					}

					err = cmd.Wait()
					ErrHandle(err)
				} else {
					inputFile, err := os.Open(tempDirectory + `/` + vodID + `.ts`)
					ErrHandle(err)

					outputFile, err := os.Create(downloadPath + `/` + vodID + `.ts`)
					ErrHandle(err)
					defer outputFile.Close()

					_, err = io.Copy(outputFile, inputFile)
					ErrHandle(err)

					err = inputFile.Close()
					ErrHandle(err)
				}
			}

			progressBar.SetValue(1)
			notify.Alert(title, "Notice", "Download Complete", dirThumb+"/"+vodID+".jpg")

			fmt.Println(FindElem(queueID, vodID))
			DelQueue(FindElem(queueID, vodID))

			ClearDir(tempDirectory)

			status.SetText(loadLang("downloadComplete"))

			OpenURL(downloadPath)
		}()
	})

	queueButton := widget.NewButtonWithIcon("", theme.MailSendIcon(), func() {
		moreInfoW := fyne.CurrentApp().NewWindow(title)

		moreInfoW.SetContent(MoreView(moreInfoW))
		moreInfoW.Resize(fyne.NewSize(500, 400))
		moreInfoW.SetIcon(theme.MoveDownIcon())
		moreInfoW.CenterOnScreen()
		moreInfoW.Show()
	})

	buttonBox := widget.NewHBox(
		layout.NewSpacer(),
		queueButton,
		button,
	)

	intervalBox := widget.NewHBox(
		intervalCheck,
		layout.NewSpacer(),
	)

	searchBox := widget.NewVBox(
		keyEntry,    // 주소 입력
		buttonBox,   // 다운로드 버튼
		intervalBox, // 구간 다운로드 버튼
	)

	homeLayoutBox := widget.NewVBox(
		layout.NewSpacer(),
		layout.NewSpacer(),
		searchBox,
		layout.NewSpacer(),
	)

	return homeLayoutBox
}

func Advanced(w2 fyne.Window) (fyne.CanvasObject, *ini.File) { // 설정
	cfg, err := ini.Load(dirBin + `/setting.ini`)
	errINI(err)

	defLang := widget.NewSelect([]string{"English", "Korean"}, func(langOption string) {
		cfg.Section("system").Key("DEFAULT_LANG").SetValue(langOption)
	})

	downOption := widget.NewSelect([]string{"Multi", "Single"}, func(c string) {
		cfg.Section("main").Key("DOWNLOAD_OPTION").SetValue(c)
	})

	defMaxConnection := widget.NewSelect([]string{"10", "100", "500", "1000"}, func(maxConNum string) {
		cfg.Section("main").Key("MAX_CONNECTION").SetValue(maxConNum)
	})

	defDownDirEntry := widget.NewEntry()

	defDownDir := widget.NewButtonWithIcon(loadLang("fileExplorer"), theme.FolderOpenIcon(), func() {
		go func() {
			selDownDir, err := dlog.Directory().Title(title).Browse()
			if err == nil {
				if len(selDownDir) != 0 {
					cfg.Section("main").Key("DOWNLOAD_DIR").SetValue(selDownDir)
					defDownDirEntry.SetText(selDownDir)
				}
			}
		}()
	})

	defSelEnc := widget.NewSelect([]string{"true", "false"}, func(enc string) {
		cfg.Section("encode").Key("ENCODING").SetValue(enc)
	})

	defSelEncType := widget.NewSelect([]string{"mp4", "avi", "mkv"}, func(encType string) {
		cfg.Section("encode").Key("ENCODING_TYPE").SetValue(encType)
	})

	saveSetting := widget.NewButtonWithIcon(loadLang("saveSetting"), theme.DocumentSaveIcon(), func() {
		err = cfg.SaveTo(dirBin + `/setting.ini`)
		ErrHandle(err)

		dialog.ShowInformation(title, loadLang("saved"), w2)
	})

	saveSettingExit := widget.NewButtonWithIcon(loadLang("exit"), theme.CancelIcon(), func() {
		w2.Close()
	})

	resetSetting := widget.NewButtonWithIcon(loadLang("resetSetting"), theme.ViewRefreshIcon(), func() {
		dialog.ShowConfirm(title, loadLang("realResetSetting"), func(b bool) {
			if b {
				err = os.Remove(dirBin + `/setting.ini`)
				ErrHandle(err)

				makeINI()

				RunAgain()
			}
		}, w2)
	})

	saveSetting.Style = widget.PrimaryButton
	saveSettingBox := widget.NewHBox(saveSetting, saveSettingExit)
	saveSettingLayout := fyne.NewContainerWithLayout(layout.NewBorderLayout(nil, nil, resetSetting, saveSettingBox), resetSetting, saveSettingBox)

	defLang.SetSelected(cfg.Section("system").Key("DEFAULT_LANG").String())
	downOption.SetSelected(cfg.Section("main").Key("DOWNLOAD_OPTION").String())
	defMaxConnection.SetSelected(cfg.Section("main").Key("MAX_CONNECTION").String())
	defDownDirEntry.SetText(cfg.Section("main").Key("DOWNLOAD_DIR").String())
	defSelEnc.SetSelected(cfg.Section("encode").Key("ENCODING").String())
	defSelEncType.SetSelected(cfg.Section("encode").Key("ENCODING_TYPE").String())

	defLang.OnChanged = func(s string) {
		dialog.ShowConfirm(title, loadLang("askRunAgainLang"), func(c bool) {
			if c {
				cfg.Section("system").Key("DEFAULT_LANG").SetValue(s)
				err = cfg.SaveTo(dirBin + `/setting.ini`)
				ErrHandle(err)

				RunAgain()
			}
		}, w2)
	}

	form := &widget.Form{}

	defLangBox := widget.NewHBox(defLang)
	defDownOptionBox := widget.NewHBox(downOption)
	defMaxConnectionBox := widget.NewHBox(defMaxConnection)
	defDownDirBox := widget.NewHBox(defDownDirEntry, defDownDir)
	defSelEncBox := widget.NewHBox(defSelEnc)
	defSelEncTypeBox := widget.NewHBox(defSelEncType)

	form.Append(loadLang("optionLanguage"), defLangBox)
	form.Append(loadLang("optionDownload"), defDownOptionBox)
	form.Append(loadLang("optionMaxConnection"), defMaxConnectionBox)
	form.Append(loadLang("optionDownloadLocation"), defDownDirBox)
	form.Append(loadLang("optionEncoding"), defSelEncBox)
	form.Append(loadLang("optionEncodingType"), defSelEncTypeBox)

	settingMenu := widget.NewVBox(
		widget.NewGroup(loadLang("defaultSetting"),
			form,
			saveSettingLayout,
		),
	)

	return settingMenu, cfg
}

func AddQueue(title, vodid, time, thumb string, prog *widget.ProgressBar, status *widget.Label, progStatus *widget.Entry, cmd *exec.Cmd) {
	fmt.Println("--- 대기열 추가")
	fmt.Println("ID: " + vodid)
	fmt.Println("Title: " + title)
	fmt.Println("Time: " + time)
	fmt.Println("Thumbnail: " + thumb)

	_, err = grab.Get(fmt.Sprintf("%s/%s.jpg", dirThumb, vodid), thumb)
	ErrHandle(err)

	// string
	queueID = append(queueID, vodid)
	queueTitle = append(queueTitle, title)
	queueTime = append(queueTime, time)
	queueThumb = append(queueThumb, fmt.Sprintf("%s/%s.jpg", dirThumb, vodid))

	// widget.ProgressBar
	queueProgress = append(queueProgress, prog)

	// widget.Entry
	queueProgStatus = append(queueProgStatus, progStatus)

	// widget.Label
	queueStatus = append(queueStatus, status)

	// exec.Cmd
	queueCmd = append(queueCmd, cmd)
}

func DelQueue(i int) {
	// string
	queueID = queueID[:i+copy(queueID[i:], queueID[i+1:])]
	queueTitle = queueTitle[:i+copy(queueTitle[i:], queueTitle[i+1:])]
	queueTime = queueTime[:i+copy(queueTime[i:], queueTime[i+1:])]
	queueThumb = queueThumb[:i+copy(queueThumb[i:], queueThumb[i+1:])]

	// widget.ProgressBar
	queueProgress = queueProgress[:i+copy(queueProgress[i:], queueProgress[i+1:])]

	// widget.Entry
	queueProgStatus = queueProgStatus[:i+copy(queueProgStatus[i:], queueProgStatus[i+1:])]

	// widget.Label
	queueStatus = queueStatus[:i+copy(queueStatus[i:], queueStatus[i+1:])]

	// exec.Cmd
	queueCmd = queueCmd[:i+copy(queueCmd[i:], queueCmd[i+1:])]
}

func MoreView(moreInfoW fyne.Window) *widget.ScrollContainer {
	queue := widget.NewGroup("대기열")

	fmt.Println(queueID)
	fmt.Println(len(queueID))
	fmt.Println(queueTitle)
	fmt.Println(queueTime)
	fmt.Println(queueThumb)

	if len(queueID) != 0 {
		for i := range queueID {
			fmt.Println("--- 대기열 로드")
			fmt.Println("Title: " + queueTitle[i])
			fmt.Println("Time: " + queueTime[i])
			fmt.Println("Thumbnail: " + queueThumb[i])

			queueVODID := queueID[i]

			queueTimeInt, err := strconv.Atoi(queueTime[i])
			ErrHandle(err)

			var stopButton *widget.Button
			stopButton = widget.NewButton(loadLang("cancel"), func() {
				stopProg := dialog.NewProgressInfinite(title, "진행 중지를 기다리는 중...", moreInfoW)
				stopProg.Show()

				switch queueProgStatus[i].Text {
				case "wait":
					stopProg.Hide()
					dialog.ShowInformation(title, loadLang("stoppedNoStatus"), moreInfoW)
				case "download":
					queueProgStatus[i].SetText("press_stop")

					for {
						if !ContainsElem(queueID, queueVODID) {
							break
						}

						time.Sleep(1 * time.Second)
					}

					stopProg.Hide()
					dialog.ShowInformation(title, loadLang("stoppedDownload"), moreInfoW)
					notify.Alert(title, "Notice", "Download Canceled", dirThumb+"/"+queueVODID+".jpg")
				case "merge":
					queueProgStatus[i].SetText("press_stop")

					for {
						if !ContainsElem(queueID, queueVODID) {
							break
						}

						time.Sleep(1 * time.Second)
					}

					stopProg.Hide()
					dialog.ShowInformation(title, loadLang("stoppedDownload"), moreInfoW)
					notify.Alert(title, "Notice", "Download Canceled", dirThumb+"/"+queueVODID+".jpg")
				case "move":
					queueProgStatus[i].SetText("press_stop")

					for {
						if !ContainsElem(queueID, queueVODID) {
							break
						}

						time.Sleep(1 * time.Second)
					}

					stopProg.Hide()
					dialog.ShowInformation(title, loadLang("stoppedDownload"), moreInfoW)
					notify.Alert(title, "Notice", "Download Canceled", dirThumb+"/"+queueVODID+".jpg")
				case "encode":
					stopProg.Hide()
					dialog.ShowInformation(title, loadLang("stoppedNoStatus"), moreInfoW)
				case "wait_incomplete_download":
					stopProg.Hide()
					dialog.ShowInformation(title, loadLang("stoppedNoStatus"), moreInfoW)
				}

				canvas.Refresh(moreInfoW.Content())
				stopButton.Disable()
			})

			if queueProgStatus[i].Text == "encode" {
				stopButton.Disable()
			}

			h := queueTimeInt / 3600
			m := (queueTimeInt - (3600 * h)) / 60
			s := queueTimeInt - (3600 * h) - (m * 60)

			moreViewForm := widget.NewVBox(
				widget.NewLabel(fmt.Sprintf("%s: %s", loadLang("vodTitle"), queueTitle[i])),
				widget.NewLabel(loadLang("vodTime")+": "+fmt.Sprintf("%d시간 %d분 %d초", h, m, s)),
				widget.NewHBox(queueStatus[i], stopButton),
			)

			vodThumbnailImg := &canvas.Image{
				File:     queueThumb[i],
				FillMode: canvas.ImageFillOriginal,
			}
			canvas.Refresh(vodThumbnailImg)

			queueInfo := fyne.NewContainerWithLayout(
				layout.NewHBoxLayout(), vodThumbnailImg, moreViewForm,
			)

			queueLayout := widget.NewVBox(
				queueInfo,
				queueProgress[i],
			)

			queue.Append(queueLayout)
		}
	} else {
		queue.Append(widget.NewLabelWithStyle(loadLang("noQueue"), fyne.TextAlignCenter, fyne.TextStyle{Bold: false}))
	}

	return widget.NewScrollContainer(queue)
}

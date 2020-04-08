package main

// TODO:
// ENCODING = false 일 경우 ts 파일을 DOWNLOAD_DIR에 생성
// regexp에 맞게 복사된 경우 자동으로 감지
// 인코딩 형식, 화질, 영상 길이 정하는 메뉴 생성

import (
  "os"
  "os/exec"
  "io"
  "io/ioutil"
  "net/http"
  "net/url"
  "path/filepath"
  "encoding/json"
  "encoding/hex"
  "crypto/rand"
  "crypto/sha256"
  "time"
  "fmt"
  "runtime"
  "sync"
  "regexp"
  "strconv"
  "strings"
  "bytes"
  "syscall"
  "math/big"
)

import (
  "github.com/lxn/walk"
. "github.com/lxn/walk/declarative"

  "github.com/dariubs/percent"
  "github.com/cavaliercoder/grab"
  "gopkg.in/ini.v1"
)

var (
  version = "1.0"
  dir_temp = os.Getenv("TEMP") + `\tmi_tips`
  dir_bin =`C:\Users\Public\Documents\tmi_tips\bin`
  API_URL = "https://dl.tmi.tips/asset/api/dl_core/"
  sha_ffmpeg = "337e5fcee11f9f7b967c12c74b70c8b24b67525843b7511a36ee82d0277536b7"
)

type TMI struct {
  Success   bool   `json:"success"`
  Link      string `json:"link"`
  Msg       string `json:"msg"`
  Code      int    `json:"code"`
}

type Status struct {
  Version   string `json:"version"`
  Reset_ini bool   `json:"reset_ini"`
  Note      string `json:"note"`
  Url       string `json:"url"`
}

func Random(i int) string {
  n, _ := rand.Int(rand.Reader, big.NewInt(int64(i)))

  return strconv.Itoa(int(n.Int64()))
}

func DownloadFile(filepath string, url string, ts_n string) {
  ts_url := url + "chunked" + "/" + ts_n + ".ts"

  status, _ := http.Get(ts_url)
  if status.StatusCode == 403 {
    ts_url := url + "chunked" + "/" + ts_n + "-muted.ts"

    grab.Get(filepath, ts_url)
  } else {
    grab.Get(filepath, ts_url)
  }
  defer status.Body.Close()
}

func ClearDir(dir string) error {
	files, err := filepath.Glob(filepath.Join(dir, "*"))
	if err != nil {
		return err
	}
	for _, file := range files {
		err = os.RemoveAll(file)
		if err != nil {
			return err
		}
	}
	return nil
}

func tsFinder(token string) int {
  resp, _ := http.Get("http://vod-secure.twitch.tv/" + token + "/chunked/index-dvr.m3u8")
  defer resp.Body.Close()

  data, _ := ioutil.ReadAll(resp.Body)

  ts := strings.Count(string(data), ".ts")

  return ts
}

func makeINI() {
  ini_file, _ := os.OpenFile(dir_bin + `\setting.ini`, os.O_CREATE|os.O_RDWR, os.FileMode(0644))
  fmt.Fprintf(ini_file, "; 아래 내용은 본 프로그램에 대해 충분히 숙지 후 수정하시기 바랍니다.\n; 지원하는 포맷 : mp4, mkv, avi, flv, wmv, ts, mov, 3gp\n\n[main]\nPARALLEL_NUM = 100\nDOWNLOAD_DIR = %s\nREMOVE_CODE_ENTER = true\n\n[update]\nCHECK_UPDATE = true\n\n[encode]\nENCODING = true\nENCODING_TYPE = mp4\n\n[misc]\nRESET_OPTION = false\nIGNORE_CLIPBOARD_NOTICE = false", os.Getenv("USERPROFILE") + `\Desktop`)

  ini_file.Close()
}

func jsonParse(url string) ([]byte) {
  req, _ := http.NewRequest("GET", url, nil)
  client := &http.Client{Timeout: time.Second * 10}
  resp, _ := client.Do(req)

  defer resp.Body.Close()

  body, _ := ioutil.ReadAll(resp.Body)

  return body
}

func checkClp(c bool, r bool) string {
  if !r {
    myWindow, _ := walk.NewMainWindow()

    clipboard, _ := walk.Clipboard().Text()
    is_matched, _ := regexp.MatchString(`\w{8}-\w{4}-\w{4}-\w{4}-\w{12}`, clipboard)

    if is_matched {
      if c {
        return clipboard
      }

      ok := walk.MsgBox(myWindow, "TMI Downloader", `클립보드에 복사된 코드가 있습니다.
사용하시겠습니까?

코드 : ` + clipboard, walk.MsgBoxYesNo)

      if ok == 6 {
        return clipboard
      } else {
        return "이 곳에 코드를 입력해주세요."
      }
    }

    myWindow.Close()
  }
  return "이 곳에 코드를 입력해주세요."
}

func key_check(cb string, mw *walk.MainWindow) string {
  clipboard, _ := walk.Clipboard().Text()
  is_matched, _ := regexp.MatchString(`\w{8}-\w{4}-\w{4}-\w{4}-\w{12}`, clipboard)

  if is_matched {
    client := &http.Client{}

    data := url.Values{}
    data.Set("key", cb)

    req, _ := http.NewRequest("POST", API_URL, bytes.NewBufferString(data.Encode()))
    req.Header.Set("Content-Type", "application/x-www-form-urlencoded; param=value")

    resp, _ := client.Do(req)

    body , _ := ioutil.ReadAll(resp.Body)
    resp.Body.Close()

    var key_data TMI
    _ = json.Unmarshal(body, &key_data)

    key_status := key_data.Success

    if !key_status {
      walk.MsgBox(mw, "TMI Downloader", key_data.Msg + `
프로그램을 다시 실행해주세요.`, walk.MsgBoxIconError)
      os.Exit(1)
    }

    vod_token := key_data.Link

    return vod_token
  } else {
    walk.MsgBox(mw, "TMI Downloader", `알맞지 않은 코드입니다.
프로그램을 다시 실행해주세요.`, walk.MsgBoxIconError)
    os.Exit(1)
    return "에러"
  }
}

func errINI(e error, mw *walk.MainWindow) {
  if e != nil {
    os.Remove(dir_bin + `\setting.ini`)
    makeINI()

    walk.MsgBox(mw, "TMI Downloader", `설정에 오류가 있어서 초기화되었습니다.
프로그램을 다시 실행해주세요.` , walk.MsgBoxIconError)
    os.Exit(1)
  }
}

func sha256Sum(t string) string {
  file, _ := os.Open(t)
  defer file.Close()

  h := sha256.New()
  io.Copy(h, file)

  return hex.EncodeToString(h.Sum(nil))
}

func checkRunType_S(c bool) string {
  if c {
    return "붙여넣기 && 다운로드"
  } else {
    return "다운로드"
  }
}

func checkRunType_V(c bool) bool {
  if c {
    return false
  } else {
    return true
  }
}

type counter struct {
  i int
  mu sync.Mutex
}

func (c *counter) increment() {
  c.mu.Lock()
  c.i += 1
  c.mu.Unlock()
}

func main() {
  wg := new(sync.WaitGroup)
  c := counter{i: 0}

  runtime.GOMAXPROCS(runtime.NumCPU())

  os.MkdirAll(dir_temp, 0777)
  os.MkdirAll(dir_bin, 0777)

  var (
    mainWindow *walk.MainWindow
    status, download_path, token_key *walk.TextEdit
    download_button *walk.PushButton
    progress_bar *walk.ProgressBar
    openAction *walk.Action
  )

  myWindow, _ := walk.NewMainWindow()

  cfg, err := ini.Load(dir_bin + `\setting.ini`)
  errINI(err, myWindow)

  // MAIN
  parralle_num, err := cfg.Section("main").Key("PARALLEL_NUM").Int()
  errINI(err, myWindow)
  dir_down := cfg.Section("main").Key("DOWNLOAD_DIR").String()
  remove_code_enter, err := cfg.Section("main").Key("REMOVE_CODE_ENTER").Bool()
  errINI(err, myWindow)

  // UPDATE
  check_update, err := cfg.Section("update").Key("CHECK_UPDATE").Bool()
  errINI(err, myWindow)

  // ENCODE
  encoding, err := cfg.Section("encode").Key("ENCODING").Bool()
  errINI(err, myWindow)
  encoding_type := cfg.Section("encode").Key("ENCODING_TYPE").String()

  // MISC
  reset_option, err := cfg.Section("misc").Key("RESET_OPTION").Bool()
  errINI(err, myWindow)
  ignore_clp_notice, err := cfg.Section("misc").Key("IGNORE_CLIPBOARD_NOTICE").Bool()
  errINI(err, myWindow)

  if reset_option {
    os.Remove(dir_bin + `\setting.ini`)
    makeINI()

    walk.MsgBox(myWindow, "TMI Downloader", `설정이 초기화되었습니다.
프로그램을 다시 실행해주세요.` , walk.MsgBoxOK)
    os.Exit(1)
  }

  myWindow.Close()

  mw := MainWindow{
    AssignTo: &mainWindow,
		Title:  "TMI Downloader",
    MinSize: Size{400, 130},
    MaxSize: Size{400, 130},
    Size: Size{50, 30},
		Layout:  VBox{},
    MenuItems: []MenuItem{
			Menu{
				Text: "설정",
				Items: []MenuItem{
					Action{
						AssignTo:    &openAction,
						Text:        "필수파일 설치",
            Enabled:     true,
						OnTriggered: func() {
              go func() {
                if !encoding {
                  walk.MsgBox(mainWindow, "TMI Downloader", `인코딩이 비활성화 상태입니다.
'설정 -> 편집'을 먼저 실행해주세요.`, walk.MsgBoxIconError)
                  return
                }
                download_button.SetEnabled(false)
                openAction.SetEnabled(false)

                progress_bar.SetRange(0, 3)
                progress_bar.SetValue(0)

                status.SetText("파일 다운로드 중... | 이 작업은 오래 걸릴 수 있습니다.")
                progress_bar.SetValue(1)
                grab.Get(dir_bin + `\ffmpeg.exe`, "https://drive.google.com/uc?export=download&id=1akNaxH6lNFQbZUMFDzE8m_Y1q0LfH1xN")

                status.SetText("파일 검증 중...")
                progress_bar.SetValue(2)

                if sha256Sum(dir_bin + `\ffmpeg.exe`) != sha_ffmpeg {
                  walk.MsgBox(mainWindow, "TMI Downloader", `파일이 손상되었습니다.
다시 설치해주세요.`, walk.MsgBoxIconError)
                  download_button.SetEnabled(true)
                  openAction.SetEnabled(true)
                  status.SetText("상태")
                  progress_bar.SetValue(0)
                  return
                }

                status.SetText("필수파일 설치 완료")
                progress_bar.SetValue(3)
                walk.MsgBox(mainWindow, "TMI Downloader", "설치 완료", walk.MsgBoxIconInformation)

                download_button.SetEnabled(true)
                openAction.SetEnabled(true)
                status.SetText("상태")
                progress_bar.SetValue(0)
              }()
            },
					},
          Action{
						Text:        "편집",
						OnTriggered: func() {
              go func() {
                status.SetText("편집 후 적용을 위해 프로그램을 다시 실행해주세요.")
                exec.Command(os.Getenv("WINDIR") + `\system32\notepad.exe`, dir_bin + `\setting.ini`).Run()
              }()
            },
					},
					Separator{},
					Action{
						Text:        "업데이트 확인",
						OnTriggered: func() {
              go func() {
                // 업데이트 가져오기
                status.SetText("업데이트 불러오는 중...")
                body := jsonParse(`https://dl.tmi.tips/bin/tmi_downloader.json`)

                var v_status Status
                json.Unmarshal(body, &v_status)

                new_version := v_status.Version
                reset_ini := v_status.Reset_ini

                if version == new_version {
                  status.SetText("이미 최신 버전입니다.")
                  return
                }

                ok := walk.MsgBox(mainWindow, "TMI Downloader", `새로운 버전 : ` + new_version + `

` + v_status.Note, walk.MsgBoxYesNo)

                if ok == 6 {
                  if reset_ini {
                    os.Remove(dir_bin + `\setting.ini`)
                    makeINI()
                  }
                  exec.Command("rundll32", "url.dll,FileProtocolHandler", v_status.Url).Start()
                }
              }()
            },
					},
				},
			},
    },
		Children: []Widget{
      PushButton{
        AssignTo: &download_button,
        Text: checkRunType_S(remove_code_enter),
        Enabled: true,
        OnClicked: func() {
          go func() {
            if parralle_num != 100 {
              ok := walk.MsgBox(mainWindow, "TMI Downloader", `편집된 설정 중에 'PARALLEL_NUM'가 있습니다.
기본값 (100)이 아닌 경우 영상에 오류가 생길 수 있으며 다운로드 중 인터넷이 느려질 수 있습니다.

그래도 ` + strconv.Itoa(parralle_num) + `으로 설정하시겠습니까?`, walk.MsgBoxYesNo)

              if ok == 7 {
                status.SetText("중지")
                return
              }
            }

            if check_update {
              // 업데이트 가져오기
              status.SetText("업데이트 불러오는 중...")
              body := jsonParse(`https://dl.tmi.tips/bin/tmi_downloader.json`)

              var v_status Status
              json.Unmarshal(body, &v_status)

              new_version := v_status.Version

              if version != new_version {
                walk.MsgBox(mainWindow, "TMI Downloader", "'설정 -> 업데이트 확인'을 먼저 실행해주세요.", walk.MsgBoxIconError)
                return
              }
            }

            if _, err := os.Stat(download_path.Text()); os.IsNotExist(err) {
              walk.MsgBox(mainWindow, "TMI Downloader", `존재하지 않는 위치입니다.
위치 : ` + download_path.Text(), walk.MsgBoxIconError)
            	return
            }

            if encoding {
              if _, err := os.Stat(dir_bin + `\ffmpeg.exe`); os.IsNotExist(err) {
                walk.MsgBox(mainWindow, "TMI Downloader", "'설정 -> 필수파일 설치'을 먼저 실행해주세요.", walk.MsgBoxIconError)
              	return
              }

              if sha256Sum(dir_bin + `\ffmpeg.exe`) != sha_ffmpeg {
                walk.MsgBox(mainWindow, "TMI Downloader", `필수파일이 손상되었습니다.
'설정 -> 필수파일 설치'을 실행해주세요.`, walk.MsgBoxIconError)
                return
              }
            }

            openAction.SetEnabled(false)
            download_button.SetEnabled(false)
            download_path.SetReadOnly(true)

            status.SetText("유효성 검증 중입니다...")
            clipboard, _ := walk.Clipboard().Text()
            vod_token := key_check(clipboard, mainWindow)

            ts_i := tsFinder(vod_token) - 1

            ClearDir(dir_temp)

            progress_bar.SetRange(0, ts_i)
            progress_bar.SetValue(0)

            g_state := 0
            d_cycle := 0

          	for i := 0; i <= ts_i; i++ {
              ts_url := "http://vod-secure.twitch.tv/" + vod_token + "/"

              i_s := strconv.Itoa(i)

              filename := dir_temp + `\` + i_s + ".ts"

              wg.Add(1)
              go func (n int){
                DownloadFile(filename, ts_url, i_s)
                c.increment()
                wg.Done()
              }(i)

              if i != 0 {
                if parralle_num > ts_i {
                  continue
                }

                if i % parralle_num == 0 {
                  d_cycle++
                  for g_state < d_cycle * parralle_num {
                    g_state := c.i

                    if g_state == 0 {
                      status.SetText("다운로드 대기 중입니다. 잠시만 기다려 주세요.")
                    } else {
                      if g_state == ( d_cycle - 1 ) * parralle_num {
                        status.SetText("대기열 추가 중입니다. 잠시만 기다려 주세요.")
                      } else {
                        status.SetText("다운로드 중... " + strconv.FormatFloat(percent.PercentOf(g_state-1, ts_i), 'f', 2, 64) + "%")
                        progress_bar.SetValue(g_state)
                        fmt.Printf("%d | %d\n", g_state, ts_i)
                      }
                      if g_state >= d_cycle * parralle_num {
                        break
                      }
                    }
                    time.Sleep(1 * time.Second)
                  }
                }
              }
            }

            for g_state < ts_i {
              g_state := c.i

              if g_state < 1 {
                status.SetText("다운로드 대기 중입니다. 잠시만 기다려 주세요.")
                time.Sleep(1 * time.Second)
                fmt.Printf("%d | %d\n", g_state, ts_i)
              } else {
                status.SetText("다운로드 중... " + strconv.FormatFloat(percent.PercentOf(g_state-1, ts_i), 'f', 2, 64) + "%")
                progress_bar.SetValue(g_state)
                fmt.Printf("%d | %d\n", g_state, ts_i)
                if g_state >= ts_i {
                  break
                }
              }

              time.Sleep(1 * time.Second)
            }

            status.SetText("완료되지 않은 다운로드 기다리는 중...")
            wg.Wait()

            status.SetText("파일 생성 중...")
            out, _ := os.OpenFile(dir_temp + `\result.ts`, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)

            for i := 0; i <= ts_i; i++ {
              i_s := strconv.Itoa(i)

              status.SetText("병합 중... " + strconv.FormatFloat(percent.PercentOf(i, ts_i), 'f', 2, 64) + "%")
              progress_bar.SetValue(i)

              filename, _ := os.Open(dir_temp + `\` + i_s + ".ts")
              io.Copy(out, filename)

              os.Remove(dir_temp + `\` + i_s + ".ts")
            }
            out.Close()

            if encoding {
              status.SetText("인코딩 중... | 이 작업은 오래 걸릴 수 있습니다.")
              cmd := exec.Command(dir_bin + `\ffmpeg.exe`, "-threads", "0", "-i", dir_temp + `\result.ts`, "-c", "copy", download_path.Text() + `\result.` + encoding_type)
              cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
              cmd.Run()

              status.SetText("파일 : result." + encoding_type + " | " + "위치 : " + download_path.Text())
            } else {
              inputFile, _ := os.Open(dir_temp + `\result.ts`)
              outputFile, _ := os.Create(download_path.Text() + `\result.ts`)
              defer outputFile.Close()
              io.Copy(outputFile, inputFile)
              inputFile.Close()

              status.SetText("파일 : result.ts | " + "위치 : " + download_path.Text())
            }

            progress_bar.SetValue(ts_i)

            ClearDir(dir_temp)

            exec.Command("rundll32", "url.dll,FileProtocolHandler", download_path.Text()).Start()
            walk.MsgBox(mainWindow, "TMI Downloader", "다운로드 완료", walk.MsgBoxIconInformation)
          }()
        },
      },

      TextEdit{
        AssignTo: &token_key,
        Text: checkClp(ignore_clp_notice, remove_code_enter),
        ReadOnly: false,
        Visible: checkRunType_V(remove_code_enter),
      },
      TextEdit{
        AssignTo: &download_path,
        Text: dir_down,
        ReadOnly: false,
      },
      TextEdit{
        AssignTo: &status,
        Text: "상태",
        ReadOnly: true,
      },

      ProgressBar{
        AssignTo: &progress_bar,
        Value: 0,
        MarqueeMode: false,
      },
    },
  }
  mw.Run()
}

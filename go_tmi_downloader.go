package main

import (
  "os"
  "io"
  "time"
  "net/http"
  "path/filepath"
  "net/url"
  "fmt"
  "runtime"
  "sync"
  "strconv"
)

import (
  "github.com/lxn/walk"
. "github.com/lxn/walk/declarative"
)

var dir = os.Getenv("TEMP") + `\tmi_tips`
var down_dir = os.Getenv("USERPROFILE") + `\Desktop`

func DownloadFile(filepath string, url string, ts_n string) error {
  ts_url := url + "chunked" + "/" + ts_n + ".ts"

  resp, err := http.Get(ts_url)
  if resp.StatusCode == 403 {
    ts_url := url + "chunked" + "/" + ts_n + "-muted.ts"

    resp, err := http.Get(ts_url)

    if err != nil {
      return err
    }
    defer resp.Body.Close()

    out, err := os.Create(filepath)
    if err != nil {
      return err
    }
    defer out.Close()

    _, err = io.Copy(out, resp.Body)
    return err
  }

  if err != nil {
    return err
  }
  defer resp.Body.Close()

  out, err := os.Create(filepath)
  if err != nil {
    return err
  }
  defer out.Close()

  _, err = io.Copy(out, resp.Body)
  return err
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

  os.MkdirAll(dir, 0777)

  scheme := os.Args[1]

  u, _ := url.Parse(scheme)

  arg, _ := url.ParseQuery(u.RawQuery)

  ts := arg["ts"][0]
  vod_token := arg["v_t"][0]

  var ts_i int
  ts_i, _ = strconv.Atoi(ts)

  var mainWindow *walk.MainWindow

  var download_status, download_path *walk.TextEdit
  var download_button *walk.PushButton
  var progress_bar *walk.ProgressBar

  MainWindow{
    AssignTo: &mainWindow,
		Title:  "TMI Downloader",
    MinSize: Size{400, 130},
    Size: Size{50, 30},
		Layout:  VBox{},
		Children: []Widget{
      PushButton{
        AssignTo: &download_button,
        Text: "다운로드",
        Enabled: true,
        OnClicked: func() {
          if _, err := os.Stat(download_path.Text()); os.IsNotExist(err) {
            download_status.SetText("존재하지 않는 위치입니다. (" + download_path.Text() + ")")
          	return
          }

          download_path.SetReadOnly(true)
          download_button.SetEnabled(false)

          ClearDir(dir)

          fmt.Println(dir)

          progress_bar.SetRange(0, ts_i)
          progress_bar.SetValue(0)

          go func () {
          	for i := 0; i <= ts_i; i++ {
              ts_url := "http://vod-secure.twitch.tv/" + vod_token + "/"

              var i_s string
              i_s = strconv.Itoa(i)

              filename := dir + `\` + i_s + ".ts"

              download_status.SetText("파일 등록 중...")
              progress_bar.SetValue(i)

              wg.Add(1)
              go func (n int){
                DownloadFile(filename, ts_url, i_s)
                c.increment()
                wg.Done()
              }(i)
            }

            progress_bar.SetValue(0)
            g_state := 0
            for g_state < ts_i {
              g_state := c.i

              var g_state_s string
              g_state_s = strconv.Itoa(g_state-1)

              if g_state == 0 {
                download_status.SetText("다운로드 대기 중입니다. 잠시만 기다려 주세요.")
              } else {
                download_status.SetText("다운로드 중... [" + g_state_s + "/" + ts + "]")
                progress_bar.SetValue(g_state)
                fmt.Println(g_state)
                if g_state == ts_i {
                  fmt.Println("break")
                  break
                }
              }

              time.Sleep(1 * time.Second)
            }

            download_status.SetText("완료되지 않은 다운로드 기다리는 중...")
            wg.Wait()

            download_status.SetText("병합 중...")
            out, _ := os.OpenFile(download_path.Text() + `\merge.ts`, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)

            for i := 0; i <= ts_i; i++ {
              var i_s string
              i_s = strconv.Itoa(i)

              progress_bar.SetValue(i)

              filename, _ := os.Open(dir + `\` + i_s + ".ts")
              io.Copy(out, filename)

              os.Remove(dir + `\` + i_s + ".ts")
            }
            out.Close()

            download_status.SetText("저장 : " + download_path.Text() + `\merge.ts`)
            progress_bar.SetValue(ts_i)

            walk.MsgBox(mainWindow, "TMI Downloader", "다운로드 완료", walk.MsgBoxIconInformation)
          }()
        },
      },

      TextEdit{
        AssignTo: &download_path,
        Text: down_dir,
        ReadOnly: false,
      },
      TextEdit{
        AssignTo: &download_status,
        Text: "상태",
        ReadOnly: true,
      },

      ProgressBar{
        AssignTo: &progress_bar,
        Value: 0,
        MarqueeMode: false,
      },
    },
  }.Run()
}

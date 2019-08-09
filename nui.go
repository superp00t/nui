package nui

import (
	"encoding/json"
	"fmt"
	"html/template"
	"image"
	"reflect"

	// need for image.Decode
	_ "image/jpeg"
	"image/png"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"runtime"
	"sync"

	"github.com/gorilla/websocket"

	"github.com/superp00t/etc"
	"github.com/superp00t/etc/yo"
	"github.com/superp00t/nui/icon"
)

type Flag uint32

const (
	None          Flag = 0
	Borderless    Flag = 1 << 1
	HideMenu      Flag = 1 << 2
	DebugRender   Flag = 1 << 3
	SilentUpdater Flag = 1 << 4
)

var (
	updated = false
)

type Settings struct {
	Title    string `json:"title"`
	Width    int    `json:"width"`
	Height   int    `json:"height"`
	URL      string `json:"-"`
	Flags    Flag   `json:"flags"`
	Icon     []byte `json:"-"`
	IconPath string `json:"icon"`
}

type Window struct {
	*http.ServeMux
	s                Settings
	c                *exec.Cmd
	electronPath     etc.Path
	ipcAddr, webAddr string
	socketL          sync.Mutex
	socket           *websocket.Conn
	OnOpen           func()
	evalPending      chan string

	bindingsL sync.Mutex
	bindings  map[string]*binding
}

func New(opts Settings) *Window {
	w := new(Window)
	w.s = opts

	w.bindings = make(map[string]*binding)
	w.ServeMux = http.NewServeMux()
	w.evalPending = make(chan string, 16)
	w.HandleFunc("/ipc", w.ipcSocket)

	return w
}

func (w *Window) HTML(s string) {
	if w.Started() {
		w.Eval(fmt.Sprintf(`document.body.innerHTML = "%s";`, template.JSEscapeString(s)))
	} else {
		if w.s.URL != "" {
			panic("nui: you cannot call HTML() when the app has not opened and you already have a URL. Remove the URL or add this function to the OnOpen event handler.")
		} else {
			w.s.URL = `data:text/html;charset=utf-8,` + url.PathEscape(s)
		}
	}
}

func (w *Window) Started() bool {
	return w.socket != nil
}

func (w *Window) Alert(s string) {
	w.Eval(fmt.Sprintf("alert(\"%s\");", template.JSEscapeString(s)))
}

func (w *Window) postJSON(v interface{}) {
	w.socketL.Lock()
	defer w.socketL.Unlock()
	if w.socket != nil {
		w.socket.WriteJSON(v)
	}
}

func (w *Window) Eval(js string) {
	w.evalPending <- js
}

func (w *Window) Minimize() {
	w.Eval("window.Minimize();")
}

func (w *Window) Maximize() {
	w.Eval("window.Minimize();")
}

func (w *Window) CSS(css string) {
	w.Eval(fmt.Sprintf(`window.RunCSS("%s");`, template.JSEscapeString(css)))
}

func (w *Window) Run() error {
	if len(w.s.Icon) == 0 {
		w.s.Icon = icon.MustAsset("ico.png")
	}

	img, _, err := image.Decode(etc.FromBytes(w.s.Icon))
	if err != nil {
		return err
	}

	switch runtime.GOOS {
	//todo: handle
	case "darwin":
	default:
		pOut := etc.NewBuffer()
		png.Encode(pOut, img)
		path := etc.TmpDirectory().Concat("nui-ico-" + etc.GenerateRandomUUID().String() + ".png")
		path.WriteAll(pOut.Bytes())
		w.s.IconPath = path.Render()
	}

	srv, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return err
	}

	_, port, _ := net.SplitHostPort(srv.Addr().String())

	w.ipcAddr = "ws://127.0.0.1:" + port + "/ipc"
	w.webAddr = "http://127.0.0.1:" + port + "/"

	if w.s.URL == "" {
		w.s.URL = w.webAddr
	} else {
		if w.s.URL[0] == '/' {
			w.webAddr = w.webAddr + w.s.URL[1:]
		} else {
			w.webAddr = w.s.URL
		}
	}

	if w.s.Flags&DebugRender != 0 {
		fmt.Println(w.webAddr)
	}

	err = w.UpdateDeps()
	if err != nil {
		return err
	}

	appFile := etc.TmpDirectory().Concat("nui-" + etc.GenerateRandomUUID().String() + ".js")
	appFile.WriteAll([]byte(w.generateJS()))

	defer appFile.Remove()
	// appEntry := w.generateJS()

	ePath := w.electronPath
	if runtime.GOOS == "windows" {
		ePath = ePath.Concat("electron.exe")
	} else {
		ePath = ePath.Concat("electron")
	}

	w.c = exec.Command(ePath.Render(), appFile.Render())
	if w.s.Flags&DebugRender != 0 {
		w.c.Stdout = os.Stdout
	}

	ch := make(chan error, 1)
	go func() {
		if err := http.Serve(srv, w); err != nil {
			ch <- err
		}
	}()

	go func() {
		ch <- w.c.Run()
	}()

	return <-ch
}

func (o Settings) on(f Flag) bool {
	return o.Flags&f != 0
}

func DisplayFatalError(err error) {
	ShowError("Fatal error", err.Error())
	yo.Fatal(err)
}

func (n *Window) Template(name, tpls string, v interface{}) {
	tpl, err := template.New(name).Parse(tpls)
	if err != nil {
		panic(err)
	}

	dat := etc.NewBuffer()
	err = tpl.Execute(dat, v)
	if err != nil {
		panic(err)
	}

	n.HTML(dat.ToString())
}

func (n *Window) Evalf(fomt string, args ...interface{}) {
	var actualArgs []interface{}
	for _, v := range args {
		vof := reflect.ValueOf(v)
		if vof.Kind() == reflect.Map || vof.Kind() == reflect.Struct {
			bytes, _ := json.Marshal(v)
			actualArgs = append(actualArgs, bytes)
			continue
		}

		switch m := v.(type) {
		case string:
			actualArgs = append(actualArgs, template.JSEscapeString(m))
		default:
			actualArgs = append(actualArgs, v)
		}
	}

	src := fmt.Sprintf(fomt, actualArgs...)

	n.Eval(src)
}

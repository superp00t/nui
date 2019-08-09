package nui

import (
	"fmt"
	"net/http"
	"reflect"
	"sync"
	"time"

	"github.com/superp00t/etc"

	"github.com/gorilla/websocket"
	"github.com/superp00t/etc/yo"
)

type ipcFrame struct {
	Command string `json:"cmd"`
	Name    string `json:"name,omitempty"`
	Method  string `json:"method,omitempty"`
	Source  string `json:"src,omitempty"`
	ID      string `json:"id,omitempty"`
	Args    Args   `json:"args,omitempty"`
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
} // use default options

func (w *Window) ipcSocket(rw http.ResponseWriter, r *http.Request) {
	c, err := upgrader.Upgrade(rw, r, nil)
	if err != nil {
		return
	}

	w.socket = c

	go func() {
		for {
			dat, ok := <-w.evalPending
			if !ok {
				return
			}
			w.postJSON(ipcFrame{
				Command: "eval",
				Source:  dat,
			})
		}
	}()

	w.evalPending <- `if (window["NUI"]) NUI();`

	if w.OnOpen != nil {
		go func() {
			time.Sleep(60 * time.Millisecond)
			w.OnOpen()
		}()
	}

	defer c.Close()
	for {
		var frame ipcFrame
		err := c.ReadJSON(&frame)
		if err != nil {
			break
		}

		switch frame.Command {
		case "api":
			w.invokeApiRequest(&frame)
		}

		yo.Spew(frame)
	}
}

type binding struct {
	sync.Mutex
	object  interface{}
	methods map[string]*methodStore
}

type methodStore struct {
	Name string
}

type Args []interface{}

func (a Args) String(arg int) string {
	if arg >= len(a) {
		return ""
	}

	s, _ := a[arg].(string)
	return s
}

func (a Args) Float64(arg int) float64 {
	if arg >= len(a) {
		return 0
	}

	s, _ := a[arg].(float64)
	return s
}

func (a Args) Int64(arg int) int64 {
	return int64(a.Float64(arg))
}

func (a Args) Bool(arg int) bool {
	if arg >= len(a) {
		return false
	}

	b, _ := a[arg].(bool)
	return b
}

func (w *Window) Close() error {
	w.postJSON(ipcFrame{
		Command: "close",
	})

	return nil
}

func (w *Window) Bind(name string, object interface{}) error {
	w.bindingsL.Lock()
	bind := new(binding)
	bind.object = object
	bind.methods = make(map[string]*methodStore)

	js := etc.NewBuffer()

	oType := reflect.TypeOf(object)

	fmt.Fprintf(js, "if (!window[\"%[1]s\"]) { window[\"%[1]s\"] = {}; }\n\n", name)
	for i := 0; i < oType.NumMethod(); i++ {
		method := oType.Method(i)
		mName := method.Name
		bind.methods[mName] = &methodStore{method.Name}

		fmt.Fprintf(js, "%s.%s = function() {\n", name, mName)
		fmt.Fprintf(js, " return _IPC_func(\"%s\", \"%s\", Array.from(arguments));\n", name, mName)
		fmt.Fprintf(js, "}\n\n")
	}

	w.evalPending <- js.ToString()

	w.bindings[name] = bind
	w.bindingsL.Unlock()

	return nil
}

func (w *Window) invokeApiRequest(frame *ipcFrame) {
	w.bindingsL.Lock()
	bind := w.bindings[frame.Name]
	if bind == nil {
		w.bindingsL.Unlock()
		return
	}
	w.bindingsL.Unlock()

	bind.Lock()
	method := bind.methods[frame.Method]
	bind.Unlock()

	mFunc := reflect.ValueOf(bind.object).MethodByName(method.Name)

	if method == nil {
		return
	}

	mType := mFunc.Type()

	sl := make([]reflect.Value, mType.NumIn())

	for x := 0; x < mType.NumIn(); x++ {
		var i interface{}
		switch mType.In(x).Kind() {
		case reflect.String:
			i = frame.Args.String(x)
		case reflect.Float64:
			i = frame.Args.Float64(x)
		case reflect.Int64:
			i = frame.Args.Int64(x)
		case reflect.Bool:
			i = frame.Args.Bool(x)
		default:
			panic("cannot handle type: " + mType.In(x).String())
		}

		sl[x] = reflect.ValueOf(i)
	}

	out := mFunc.Call(sl)
	result := make([]interface{}, len(out))
	for x := range result {
		result[x] = out[x].Interface()
	}

	w.postJSON(ipcFrame{
		Command: "api-result",
		ID:      frame.ID,
		Args:    result,
	})
}

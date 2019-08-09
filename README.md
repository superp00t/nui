# nui (WIP)

![img](winmanifest/nui.ico)

nui (naive ui) is a library for creating Go desktop applications with GUIs defined in HTML/CSS/JS.

It works by downloading a copy of Electron on your computer and controlling it through a web server, rather than binding WebKit directly.

nui should work on most major desktop environments.

On Windows it requires a Common Control version 6 manifest, so import ours if you don't already have one: 

```go
import _ "github.com/superp00t/nui/winmanifest"
```

# JS->Go RPC calls

To generate a Go binding, first create a model such as this.

```go
type AppModel struct {
  win *Window
}

func (app *AppModel) HandleClick() {
  a.win.Alert("Button clicked!")
}

func (app *AppModel) Ints() ([]int, bool) {
  return []int{1,2,3,4,5}, true
}
```

Generating the binding:

```go
app := new(AppModel)
app.win = win

// Generates JavaScript interface
win.Bind("app", app)
```

This will expose AppModel in your nui Electron window as an object in your Window's JavaScript instance:
```js

// invoked after binding is complete
window.NUI = async () => {
  // ints = [ [1,2,3,4,5], true ] 
  let ints = await app.Ints(); 
  console.log(typeof ints); // array
  console.log(typeof ints[0]); // array
  console.log(typeof ints[0][0]); // number
}
```

This pattern *should* work with most JSON-serializable return values.

## Credits

- nui is inspired heavily by [zserge/webview](https://github.com/zserge/webview), and shares much of the same API as it
- The Electron updater GUI is provided by [andlabs/ui](https://github.com/andlabs/ui)
- nui also is inspired by [astilectron.](https://github.com/asticode/go-astilectron)
- Incorporates [Electron.](https://electronjs.org/)
- Uses the GitHub API to download Electron.


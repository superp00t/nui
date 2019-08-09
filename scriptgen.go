package nui

import (
	"encoding/json"
	"fmt"
)

const jsTemplate = `
const {app, BrowserWindow, ipcMain} = require("electron");
const process = require("process");

var mw = null;

ipcMain.on("close", () => {
	process.exit();
});

ipcMain.on("log", (_, data) => {
	console.log.apply(console, data);
});

app.on(
"ready",
() => {
	console.log("Application ready");	
	var opts = %s;
	
	opts.webPreferences = {
		nodeIntegration: true,
		webSecurity:     false
	};

	if (opts.flags & (1 << 1))
		opts.frame = false;

	mw = new BrowserWindow(opts);
	mw.loadURL("%s");
	if (opts.flags & (1 << 2))
		mw.setMenu(null);

	mw.webContents.executeJavaScript(%s);
});
`

const shellSrc = `const {ipcRenderer, remote} = require("electron");

remote.globalShortcut.register('CommandOrControl+Shift+K', () => {
  remote.BrowserWindow.getFocusedWindow().webContents.openDevTools()
})

window.addEventListener('beforeunload', () => {
  remote.globalShortcut.unregisterAll()
})

window.Minimize = function() {
	remote.getCurrentWindow().minimize();
}

window.Maximize = function() {
	remote.getCurrentWindow().maximize();
}

window.RunCSS = function(css) {
	var style = document.createElement('style');
	var head = document.head || document.getElementsByTagName('head')[0];
	style.setAttribute('type', 'text/css');
	if (style.styleSheet) {
		style.styleSheet.cssText = css;
	} else {
		style.appendChild(document.createTextNode(css));
	}
	head.appendChild(style);
}

function Log() {
	ipcRenderer.send("log", Array.from(arguments));
}

window.GenUUID = function() {
	return 'xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx'.replace(/[xy]/g, function(c) {
		var r = Math.random() * 16 | 0, v = c == 'x' ? r : (r & 0x3 | 0x8);
		return v.toString(16);
	});
}

window._IPC_cbs = {};

window._IPC_func = function(object, method, args) {
	return new Promise(function(y, n) {
		var uid = GenUUID();
		_IPC_cbs[uid] = function(results) {
			y(results);
			delete _IPC_cbs[uid];
		}

		_IPC.send(JSON.stringify({
			cmd: "api",
			name: object,
			method: method,
			args: args,
			id: uid
		}));
	});
}

delete console["log"];
delete console["warn"];
console.log = Log;
console.warn = Log;

window.CloseApp = function() {
	ipcRenderer.send('close', true);
}

window._IPC = new WebSocket("%s");
_IPC.onmessage = function(event) {
	var data = JSON.parse(event.data);

	switch (data["cmd"]) {
	case "eval":
	console.log("evaluating", data.src);
	try {
		eval(data.src);
	} catch(err) {
		alert(err.message + ": " + data.src);
	}
	break;

	case "api-result":
	_IPC_cbs[data["id"]](data["args"]);
	break;

	case "close":
	CloseApp();
	break;
	} 
}

_IPC.onerror = function(err) {
	console.log(JSON.stringify(err));
	console.log("IPC socket error " + err.name + " " + err.message);
}

_IPC.onclose = function() {
	CloseApp();
}
`

func (w *Window) generateJS() string {
	shell, _ := json.Marshal(fmt.Sprintf(shellSrc, w.ipcAddr))
	data, _ := json.Marshal(w.s)
	dat := fmt.Sprintf(jsTemplate, data, w.webAddr, shell)
	return dat
}

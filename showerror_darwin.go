// +build darwin

package nui

func ShowError(title, data string) {
	exec.Command("osascript", "-e" "tell app \"Finder\" to display dialog '" + data + "'").Run()
}
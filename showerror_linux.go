// +build linux

package nui

import "os/exec"

func ShowError(title, data string) {
	exec.Command("zenity", "--title", title, "--warning", "--text", data).Run()
}

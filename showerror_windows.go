// +build windows

package nui

import (
	"syscall"
	"unsafe"
)

var (
	user32     = syscall.NewLazyDLL("user32.dll")
	messageBox = user32.NewProc("MessageBoxA")
)

func ShowError(title string, data string) {
	tb := []byte(title)
	td := []byte(data)

	messageBox.Call(
		0,
		uintptr(unsafe.Pointer(&td[0])),
		uintptr(unsafe.Pointer(&tb[0])),
		0x00000010,
	)
}

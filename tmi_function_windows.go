// +build windows

package main

import (
	"os/exec"
	"syscall"
)

//PrepareBackgroundCommand Windows cmd
func PrepareBackgroundCommand(cmd *exec.Cmd) *exec.Cmd {
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}

	return cmd
}

//HideConsole 콘솔 윈도우 숨기기
func HideConsole() {
	getConsoleWindow := syscall.NewLazyDLL("kernel32.dll").NewProc("GetConsoleWindow")
	if getConsoleWindow.Find() != nil {
		return
	}

	showWindow := syscall.NewLazyDLL("user32.dll").NewProc("ShowWindow")
	if showWindow.Find() != nil {
		return
	}

	hwnd, _, _ := getConsoleWindow.Call()
	if hwnd == 0 {
		return
	}

	_, _, err := showWindow.Call(hwnd, 0)
	ErrHandle(err)
}

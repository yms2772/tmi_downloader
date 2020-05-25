// +build !windows

package main

import (
	"os/exec"
)

//PrepareBackgroundCommand MacOS cmd
func PrepareBackgroundCommand(cmd *exec.Cmd) *exec.Cmd {
	return cmd
}

//HideConsole 콘솔 윈도우 숨기기
func HideConsole() {

}

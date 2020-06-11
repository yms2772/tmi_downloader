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

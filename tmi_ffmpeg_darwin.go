// +build !windows

package main

import "os/exec"

func PrepareBackgroundCommand(cmd *exec.Cmd) *exec.Cmd {
	return cmd
}

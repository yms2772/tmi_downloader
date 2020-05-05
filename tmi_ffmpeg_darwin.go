// +build !windows

package main

import "os/exec"

//PrepareBackgroundCommand MacOS cmd
func PrepareBackgroundCommand(cmd *exec.Cmd) *exec.Cmd {
	return cmd
}

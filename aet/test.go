package main

import (
	"os"
	"os/exec"
)

func runTestsCommand() {
	goTest := []string{"go", "test"}
	goTest = append(goTest, flags.Args()[1:]...)
	runCmd(goTest, func(c *exec.Cmd) {
		c.Env = appendToPathList(os.Environ(), "GOPATH", appengineDir)
	})
}

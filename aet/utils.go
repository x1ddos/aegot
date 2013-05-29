package main

import (
	"io"
	"log"
	"os"
	"os/exec"
	"strings"
)

func runCmd(args []string, f func(*exec.Cmd)) {
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	if f != nil {
		f(cmd)
	}
	if err := cmd.Run(); err != nil {
		log.Fatal(err)
	}
}

func appendToPathList(env []string, key, val string) []string {
	var (
		existingVal string
		idx         = -1
	)
	keyPrefx := key + "="
	for i, ev := range env {
		if strings.HasPrefix(ev, keyPrefx) {
			idx = i
			existingVal = ev[len(keyPrefx):]
			break
		}
	}
	if idx >= 0 {
		env[idx] = keyPrefx + existingVal + string(os.PathListSeparator) + val
	} else {
		env = append(env, keyPrefx+val)
	}
	return env
}

func isExist(path string) bool {
	_, err := os.Stat(path)
	return err == nil || !os.IsNotExist(err)
}

func copyFile(src, dst string) (err error) {
	s, err := os.Open(src)
	if err != nil {
		return err
	}
	d, err := os.Create(dst)
	if err != nil {
		return err
	}
	_, err = io.Copy(d, s)
	return
}

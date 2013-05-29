package main

import (
	"fmt"
	"os"
	"log"
	"os/exec"
	"strings"
	"path/filepath"
	"net/http"
	"io/ioutil"
)

const (
	patchesUrl = "https://raw.github.com/crhym3/aegot/master/patches"
	defaultRepoUrl = "https://code.google.com/p/appengine-go/"
	defaultVer = "1.8.0"
	// Revision, url and dest dir are appended in parseArgs().
	// Complete cmd will look like this: "hg clone -u rev URL appengineDir"
	defaultCloneCmd = "hg clone -u"
	defaultUpdateCmd = "hg update -r"
)

var (
	repoUrl string
	repoRev string
	cloneCmd string
	updateCmd string

	// Relative to "appengineDir/src"
	api_dev_dot_go    = filepath.Join("appengine_internal", "api_dev.go")

	// App Engine Go release version => repo revision map.
	// These are taken from defaultRepoUrl.
	releaseToRev = map[string]string{
		"1.8.0": "adcd6a11ae10",
	}
)

func initSourcesCommand() {
	checkInitSrcArgs()
	if err := os.MkdirAll(appengineDir, 0755); err != nil {
		log.Fatal(err)
	}

	srcDir := filepath.Join(appengineDir, "src")
	api_dev_dot_go = filepath.Join(srcDir, api_dev_dot_go)

	patchedFile := fmt.Sprintf("api_dev_%s.go", repoRev)
	ch := make(chan interface{}, 1)
	go fetchPatch(patchedFile, ch)

	// clone appengine-go repo
	var (
		cmd []string
		f func(*exec.Cmd)
	)
	if isExist(srcDir) {
		cmd = []string{updateCmd, repoRev}
		f = func(c *exec.Cmd) {
			c.Dir = srcDir
		}
	} else {
		cmd = []string{cloneCmd, repoRev, repoUrl, srcDir}
	}
	cmd = strings.Split(strings.Join(cmd, " "), " ")
	log.Print(cmd)
	runCmd(cmd, f)

	// "patch" api_dev.go
	log.Printf("Patching %s with %s", api_dev_dot_go, patchedFile)
	val := <- ch
	if err, isErr := val.(error); isErr {
		log.Fatal(err)
	}
	file, err := os.Create(api_dev_dot_go)
    if err != nil {
        log.Fatal(err)
    }
    if _, err := file.Write(val.([]byte)); err != nil {
		log.Fatal(err)
	}

	// build appengine-go packages to speed up later tests
	if flags.NArg() > 1 {
		goTestInstall := []string{"go", "test", "-i"}
		goTestInstall = append(goTestInstall, flags.Args()[1:]...)
		log.Print(goTestInstall)
		runCmd(goTestInstall, func(c *exec.Cmd){
			c.Env = appendToPathList(os.Environ(), "GOPATH", appengineDir)
		})
	}

	log.Print("Done! Try running 'make test'.")
}

func checkInitSrcArgs() {
	if strings.ContainsRune(repoRev, '.') {
		var ok bool
		if repoRev, ok = releaseToRev[repoRev]; !ok {
			log.Fatal("Unknown SDK release version")
		}
	} else {
		var found bool
		for _, v := range releaseToRev {
			if v == repoRev {
				found = true
				break
			}
		}
		if !found {
			log.Printf("WARNING: Unknown revision %q", repoRev)
		}
	}
}

func fetchPatch(patchName string, c chan interface{}) {
	url := patchesUrl + patchName
	resp, err := http.Get(url)
	if err != nil {
		c <- err
		return
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		c <- err
		return
	}
	c <- body
}

func init() {
	flags.StringVar(&repoUrl, "url", defaultRepoUrl,
		"appengine-go project repository URL")
	flags.StringVar(&repoRev, "rev", defaultVer,
		"App Engine release version or repo revision; required for init")
	flags.StringVar(&cloneCmd, "c", defaultCloneCmd,
		"command to clone the repo; don't specify rev, url or d here")
	flags.StringVar(&updateCmd, "uc", defaultUpdateCmd,
		"command to update previously clonned repo")
}

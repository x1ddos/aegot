package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const (
	patchesRoot    = "https://raw.github.com/crhym3/aegot/master/patches/"
	defaultRepoUrl = "https://code.google.com/p/appengine-go/"
	defaultVer     = "1.8.0"
	// Revision, url and dest dir are appended in parseArgs().
	// Complete cmd will look like this: "hg clone -u rev URL appengineDir"
	defaultCloneCmd  = "hg clone -u"
	defaultUpdateCmd = "hg update -r"
)

type singlePatch struct {
	// relative to patchesRoot
	src string
	// relative to "appengineDir/src"
	dst   string
	bytes []byte
}

type patchSet struct {
	patches []*singlePatch
	// revision number from repoUrl
	rev string
}

var (
	repoUrl   string
	repoRev   string
	cloneCmd  string
	updateCmd string

	// App Engine Go release version => patchset map.
	// Revisions here are taken from defaultRepoUrl.
	patchesMap = map[string]*patchSet{
		"1.8.0": {
			rev: "adcd6a11ae10",
			patches: []*singlePatch{
				{src: "api_dev.go", dst: "appengine_internal/api_dev.go"},
				{src: "internal.go", dst: "appengine_internal/internal.go"},
			},
		},
	}
)

func initSourcesCommand() {
	ps, err := findPatchSet(repoRev)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("appengine-go dir: %s", appengineDir)

	if err = os.MkdirAll(appengineDir, 0755); err != nil {
		log.Fatal(err)
	}

	c := make(chan *singlePatch, len(ps.patches))
	errc := make(chan error, len(ps.patches))
	for _, sp := range ps.patches {
		go fetchPatch(sp, ps.rev, c, errc)
	}

	// clone appengine-go repo
	var (
		srcDir = filepath.Join(appengineDir, "src")
		cmd    []string
		f      func(*exec.Cmd)
	)
	if isExist(srcDir) {
		cmd = []string{updateCmd, ps.rev}
		f = func(c *exec.Cmd) {
			c.Dir = srcDir
		}
	} else {
		cmd = []string{cloneCmd, ps.rev, repoUrl, srcDir}
	}
	cmd = strings.Split(strings.Join(cmd, " "), " ")
	log.Print(cmd)
	runCmd(cmd, f)

	// "patch" files
	count := 0
	for {
		var sp *singlePatch
		select {
		case err = <-errc:
			log.Fatal(err)
		case sp = <-c:
			log.Printf("Patching %s with %s", sp.dst, sp.src)
			dst := filepath.Join(appengineDir, "src", sp.dst)
			file, err := os.Create(dst)
			if err != nil {
				log.Fatal(err)
			}
			if _, err := file.Write(sp.bytes); err != nil {
				log.Fatal(err)
			}
		}
		count += 1
		if count == len(ps.patches) {
			break
		}
	}

	msg := "Done!"

	// build appengine-go packages to speed up later tests
	if flags.NArg() > 1 {
		goTestInstall := []string{"go", "test", "-i"}
		goTestInstall = append(goTestInstall, flags.Args()[1:]...)
		log.Print(goTestInstall)
		runCmd(goTestInstall, func(c *exec.Cmd) {
			c.Env = appendToPathList(os.Environ(), "GOPATH", appengineDir)
		})
		msg += (" Try running 'aet test -v " +
			strings.Join(flags.Args()[1:], " ") + "'")
	}

	log.Print(msg)
}

// rev can be either GAE release version (e.g. "1.8.0")
// or a commit revision (e.g. "adcd6a11ae10").
func findPatchSet(rev string) (*patchSet, error) {
	if strings.ContainsRune(rev, '.') {
		if ps, ok := patchesMap[rev]; !ok {
			return nil, fmt.Errorf("Unknown SDK release version %q", rev)
		} else {
			return ps, nil
		}
	}

	for _, v := range patchesMap {
		if v.rev == rev {
			return v, nil
		}
	}

	return nil, fmt.Errorf("Unknown revision number %q", rev)
}

func fetchPatch(sp *singlePatch, rev string, c chan *singlePatch, errc chan error) {
	url := patchesRoot + rev + "_" + sp.src
	resp, err := http.Get(url)
	if err != nil {
		errc <- err
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		errc <- fmt.Errorf("Bad response code (%d) from %s", resp.StatusCode, url)
		return
	}

	if sp.bytes, err = ioutil.ReadAll(resp.Body); err != nil {
		errc <- err
		return
	}
	c <- sp
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

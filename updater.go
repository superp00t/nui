package nui

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime"
	"sync"
	"time"

	"github.com/hashicorp/go-version"
	"github.com/superp00t/etc"
	"github.com/superp00t/etc/yo"
)

const (
	pBarResolution = 100
)

type updater struct {
	sync.Mutex
	show              bool
	text              string
	downloadSize      int64
	downloadCompleted int64
	proxy             io.ReadCloser
}

func (u *updater) Read(b []byte) (int, error) {
	i, err := u.proxy.Read(b)

	if i >= 0 {
		u.downloadCompleted += int64(i)
		u.updateProgress()
	}

	return i, err
}

func (u *updater) SetText(text string) {
	u.Lock()
	u.text = text
	u.Unlock()
}

func (u *updater) updateProgress() {
	if u.show {
		u.Lock()
		pct := ((float64(u.downloadCompleted) / float64(u.downloadSize)) * pBarResolution)
		fmt.Print("\r")
		for x := 0; x < 255; x++ {
			fmt.Print(" ")
		}
		fmt.Print("\r")
		fmt.Printf("%.2f %s", pct, u.text)
		u.Unlock()
	}
}

func ext() string {
	if runtime.GOOS == "windows" {
		return ".exe"
	}

	return ""
}

func (wind *Window) UpdateDeps() error {
	var electronPath etc.Path

	switch runtime.GOOS {
	case "windows":
		electronPath = etc.Root().Concat("ProgramData", "NUI-Electron")
	default:
		electronPath = etc.LocalDirectory().Concat("NUI-Electron")
	}

	wind.electronPath = electronPath

	binPath := electronPath.Concat("electron" + ext())

	if updated {
		return nil
	}

	updated = true

	u := &updater{}
	c := &http.Client{
		Timeout: 8 * time.Second,
	}

	u.show = !wind.s.on(SilentUpdater)

	if binPath.IsExtant() {
		f, err := os.OpenFile(binPath.Render(), os.O_RDWR, 0700)
		if err != nil {
			return err
		}

		f.Close()
	}

	var currentVersion string

	var gr GithubReleases

	// Get latest version
	h, err := c.Get("https://api.github.com/repos/electron/electron/releases/latest")
	if err != nil {
		yo.Warn(err)
	} else {
		j := json.NewDecoder(h.Body)
		err := j.Decode(&gr)
		if err != nil {
			return err
		}

		if len(gr.TagName) == 0 {
			return fmt.Errorf("nui updater: invalid tag name from GitHub")
		}

		currentVersion = gr.TagName[1:]
	}

	// Can't update if we're not able to get the current version.
	if currentVersion == "" {
		// However, it's a problem if we don't have Electron installed.
		if electronPath.IsExtant() == false {
			return fmt.Errorf("nui updater: Could not install Electron. The GitHub network appears to be currently inaccessible.")
		}

		return nil
	}

	cvsn, err := version.NewVersion(currentVersion)
	if err != nil {
		return err
	}

	var zipFile string
	var install bool

	// Electron is already installed, check for updates
	if electronPath.IsExtant() {
		vsn, err := electronPath.Concat("version").ReadAll()
		if err != nil {
			install = true
			goto doInstall
		}

		v1, err := version.NewVersion(string(vsn))
		if err != nil {
			return err
		}

		if v1.LessThan(cvsn) {
			removeContents(electronPath.Render())
			install = true
		} else {
			return nil
		}
	} else {
		electronPath.MakeDir()
		install = true
	}

doInstall:
	if !install {
		return nil
	}

	zipFile, err = getReleasePath(currentVersion)
	if err != nil {
		return err
	}

	var clength int64

	for _, s := range gr.Assets {
		if s.BrowserDownloadURL == zipFile {
			clength = s.Size
			break
		}
	}

	outpath := etc.TmpDirectory().Concat("electron-"+etc.GenerateRandomUUID().String()).Render() + ".zip"
	elFile, err := etc.FileController(outpath)
	if err != nil {
		return err
	}

	if clength <= 0 {
		err := fmt.Errorf("no file")
		return err
	}

	download, err := c.Get(zipFile)
	if err != nil {
		return err
	}

	u.downloadSize = clength
	u.proxy = download.Body

	io.Copy(elFile, u)
	elFile.Close()

	u.unzip(outpath, electronPath.Render())
	os.Remove(outpath)

	return nil
}

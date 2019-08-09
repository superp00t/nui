package main

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/superp00t/etc/yo"
)

func osExt() string {
	if os.Getenv("GOOS") == "windows" {
		return ".exe"
	}

	if os.Getenv("GOOS") == "" && runtime.GOOS == "windows" {
		return ".exe"
	}

	return ""
}

func getBuildOutput() string {
	s := os.Args[1]

	if strings.Contains(s, "/") {
		spl := strings.Split(s, "/")
		return spl[len(spl)-1] + osExt()
	}

	fl := strings.Split(s, ".")
	return fl[0] + osExt()
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("nui-build <package or file>")
		os.Exit(1)
	}

	args := []string{"build"}

	flags := `-H=windowsgui -s -w`

	args = append(args, "-ldflags", flags)

	args = append(args, "-o", getBuildOutput(), os.Args[1])

	yo.Spew(args)

	c := exec.Command("go", args...)
	c.Env = os.Environ()
	c.Stdout = os.Stdout
	c.Stdin = os.Stdin
	c.Stderr = os.Stderr
	if err := c.Run(); err != nil {
		yo.Fatal(err)
	}

	c = exec.Command("upx", getBuildOutput())
	c.Env = os.Environ()
	c.Stdout = os.Stdout
	c.Stdin = os.Stdin
	c.Stderr = os.Stderr
	c.Run()
}

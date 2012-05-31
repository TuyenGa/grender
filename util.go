package main

import (
	"path"
	"path/filepath"
	"strings"
	. "github.com/peterbourgon/bonus/xlog"
	"os/exec"
)

func RecursiveCopy(srcDir, dstDir string) {
	// Command("cp", "-r", srcDir+"/*", dstDir) doesn't shell-expand the *.
	files, err := filepath.Glob(srcDir + "/*")
	if err != nil {
		Problemf("%s", err)
		return
	}
	for _, srcFile := range files {
		cmd := exec.Command("cp", "-r", srcFile, dstDir)
		err := cmd.Run()
		if err != nil {
			Problemf("%s: %s", strings.Join(cmd.Args, " "), err)
			continue
		}
	}
}

func ShouldDescend(dir, pageFile string) bool {
	d, pf := TokenizePath(dir), TokenizePath(pageFile)
	if len(d) >= len(pf) {
		return false
	}
	if equal(pf[:len(d)], d) {
		return true
	}
	return false
}

func Subpath(rootDir, file string) string {
	d, err := filepath.Abs(rootDir)
	if err != nil {
		Problemf("Subpath(%s, %s): %s", rootDir, file, err)
		return ""
	}
	f, err := filepath.Abs(file)
	if err != nil {
		Problemf("Subpath(%s, %s): %s", rootDir, file, err)
		return ""
	}
	if strings.Index(f, d) != 0 {
		Problemf("Subpath(%s, %s): file not under directory", rootDir, file)
		return ""
	}
	return f[len(d)+1:]
}

func TokenizePath(s string) []string {
	p, a := s, []string{}
	for {
		p = path.Clean(p)
		l, r := path.Dir(p), path.Base(p)
		if r == "." || r == "/" {
			break
		}
		a = append(a, r)
		p = l
	}
	reverse(a)
	return a
}

func SplitAtExtension(file string) (string, string) {
	extLen := len(path.Ext(file))
	dotPos := len(file) - extLen
	return file[:dotPos], file[dotPos:]
}

func StripExtension(file string) string {
	base, _ := SplitAtExtension(file)
	return base
}

func equal(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := 0; i < len(a); i++ {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func reverse(a []string) {
	for i, j := 0, len(a)-1; i < j; i, j = i+1, j-1 {
		a[i], a[j] = a[j], a[i]
	}
}

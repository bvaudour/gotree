package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"time"
)

var (
	fAll     *bool
	fOnlyDir *bool
	fDepth   *int
	fUser    *bool
	fSize    *bool
	fHuman   *bool
	fMdate   *bool
	fNoColor *bool
	fHelp    *bool
	fVersion *bool

	totalDirs  int
	totalFiles int
	totalSize  int64
)

const (
	kilo int64 = 1 << 10
	mega       = kilo << 10
	giga       = mega << 10
	tera       = giga << 10
	
	VERSION = "0.2"
)

func formatArgs() []string {
	var args []string
	for _, e := range os.Args[1:] {
		if e != "--help" && strings.HasPrefix(e, "-") && len(e) > 2 {
			for _, c := range e[1:] {
				args = append(args, string([]rune{'-', c}))
			}
		} else {
			args = append(args, e)
		}
	}
	return args
}

func setFlags() {
	fAll = flag.Bool("a", false, "Display all files")
	fOnlyDir = flag.Bool("d", false, "Display only directories")
	fDepth = flag.Int("L", 3, "Max depth to explore - infinite if 0")
	fUser = flag.Bool("u", false, "Display owner")
	fSize = flag.Bool("s", false, "Display size in bytes")
	fHuman = flag.Bool("h", false, "Display size in human format")
	fMdate = flag.Bool("D", false, "Display last modified date")
	fNoColor = flag.Bool("n", false, "Doesn't display colors")
	fHelp = flag.Bool("-help", false, "Print this help")
	fVersion = flag.Bool("v", false, "Printe the version")
}

func setOptions() {
	SetMaxDepth(*fDepth)
	SetOnlyDirs(*fOnlyDir)
	SetHidden(*fAll)
	SetDirFirst(true)
}

func human(size int64) string {
	h, u := float64(size), "B"
	switch {
	case size > tera:
		h /= float64(tera)
		u = "T"
	case size > giga:
		h /= float64(giga)
		u = "G"
	case size > mega:
		h /= float64(mega)
		u = "M"
	case size > kilo:
		h /= float64(kilo)
		u = "K"
	default:
		return fmt.Sprintf("%6d%s", size, u)
	}
	return fmt.Sprintf("%4.1f%s", h, u)
}

func date(t time.Time) string {
	return fmt.Sprintf("%4d.%02d.%02d %02d:%02d", t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute())
}

func printTree(t *Tree) {
	p := t.Prefix()
	n := t.Name()
	i := t.Perm()
	if *fUser {
		i = fmt.Sprintf("%s %10s", i, t.Owner())
	}
	if *fHuman {
		i = fmt.Sprintf("%s %s", i, human(t.Size()))
	} else if *fSize {
		i = fmt.Sprintf("%s %10d", i, t.Size())
	}
	if *fMdate {
		i = fmt.Sprintf("%s %s", i, date(t.Mtime()))
	}
	if !*fNoColor {
		if t.IsDir() {
			n = fmt.Sprintf("\033[1;34m%s\033[m", n)
		} else if t.IsSymlink() {
			n = fmt.Sprintf("\033[1;36m%s\033[m", n)
		} else if t.IsExec() {
			n = fmt.Sprintf("\033[1;32m%s\033[m", n)
		}
	}
	fmt.Printf("%s[%s] %s\n", p, i, n)
}

func main() {
	setFlags()
	flag.CommandLine.Parse(formatArgs())
	setOptions()
	args := flag.Args()
	if *fHelp {
		fmt.Println(flag.ErrHelp)
		return
	} else if *fVersion {
		fmt.Println(VERSION)
		return
	}
	if len(args) == 0 {
		args = []string{"."}
	}
	for _, p := range args {
		t := NewTree(p)
		if t != nil {
			totalFiles += t.NbFiles()
			totalDirs += t.NbDirs()
			totalSize += t.Size()
			for e := range t.Iterator() {
				printTree(e)
			}
		}
	}
	size := strings.TrimSpace(human(totalSize))
	fmt.Printf("\n%s used in %d directories, %d files\n", size, totalDirs, totalFiles)
}

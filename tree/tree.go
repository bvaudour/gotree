package tree

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"os/user"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"time"
)

const (
	entry     = "├── "
	lastentry = "└── "
	cont      = "│   "
	space     = "    "
)

var (
	level     = 0
	only_dir  = false
	hidden    = false
	dir_first = false
)

func SetMaxDeep(d int) {
	if d <= 0 {
		level = 0
	} else {
		level = d
	}
}

func SetOnlyDirs(b bool) {
	only_dir = b
}

func SetHidden(b bool) {
	hidden = b
}

func SetDirFirst(b bool) {
	dir_first = b
}

type fsorter struct {
	files []os.FileInfo
}

func (f *fsorter) Len() int { return len(f.files) }
func (f *fsorter) Less(i, j int) bool {
	f1, f2 := f.files[i], f.files[j]
	if f1.IsDir() != f2.IsDir() {
		return f1.IsDir()
	}
	return f1.Name() < f2.Name()
}
func (f *fsorter) Swap(i, j int) { f.files[i], f.files[j] = f.files[j], f.files[i] }

func sortDirFirst(files []os.FileInfo) {
	f := &fsorter{files}
	sort.Sort(f)
}

func fh(f os.FileInfo) bool  { return !strings.HasPrefix(f.Name(), ".") }
func fd(f os.FileInfo) bool  { return f.IsDir() }
func fdh(f os.FileInfo) bool { return fd(f) && fh(f) }

func getFilter() func(os.FileInfo) bool {
	if only_dir {
		if hidden {
			return fd
		}
		return fdh
	} else if hidden {
		return func(os.FileInfo) bool { return true }
	}
	return fh
}

func applyFilter(files []os.FileInfo, f func(os.FileInfo) bool) (filtered []os.FileInfo) {
	for _, e := range files {
		if f(e) {
			filtered = append(filtered, e)
		}
	}
	return
}

type Tree struct {
	path      string
	file      os.FileInfo
	level     int
	levelCont []int
	idx       int
	parent    *Tree
	childs    []*Tree
	size      int64
	nbDir     int
	nbFile    int
}

func (t *Tree) isLast() bool {
	if t.parent == nil {
		return true
	}
	return t.idx == len(t.parent.childs)-1
}

func (t *Tree) IsDir() bool {
	return t.file.IsDir()
}

func (t *Tree) IsSymlink() bool {
	return t.file.Mode()&os.ModeSymlink == os.ModeSymlink
}

func (t *Tree) Size() int64 {
	return t.size
}

func (t *Tree) Mtime() time.Time {
	return t.file.ModTime()
}

func (t *Tree) Perm() string {
	return t.file.Mode().String()
}

func (t *Tree) Owner() string {
	uid := fmt.Sprint(t.file.Sys().(*syscall.Stat_t).Uid)
	if u, e := user.LookupId(uid); e == nil {
		return u.Username
	}
	return uid
}

func (t *Tree) Path() string {
	return t.path
}

func (t *Tree) Name() string {
	if t.level == 0 {
		return t.path
	}
	return t.file.Name()
}

func (t *Tree) Prefix() string {
	b := new(bytes.Buffer)
	if t.level != 0 {
		l := 0
		for _, i := range t.levelCont {
			b.WriteString(strings.Repeat(space, l-i) + cont)
			l = i + 1
		}
		b.WriteString(strings.Repeat(space, t.level-l))
		if t.isLast() {
			b.WriteString(lastentry)
		} else {
			b.WriteString(entry)
		}
	}
	return b.String()
}

func (t *Tree) NbDirs() int {
	return t.nbDir
}

func (t *Tree) NbFiles() int {
	return t.nbFile
}

func NewTree(path string) *Tree {
	f, e := os.Lstat(path)
	if e != nil {
		return nil
	}
	filter := getFilter()
	if !filter(f) {
		return nil
	}
	t := new(Tree)
	t.path = path
	t.file = f
	t.update()
	return t
}

func newSubTree(parent *Tree, f os.FileInfo, idx int, lcont []int) *Tree {
	t := new(Tree)
	t.path = filepath.Join(parent.path, f.Name())
	t.file = f
	t.level = parent.level + 1
	t.levelCont = lcont
	t.idx = idx
	t.parent = parent
	t.update()
	return t
}

func (t *Tree) update() {
	if t.IsDir() {
		if level == 0 || t.level < level-1 {
			t.subtree()
		}
		t.nbDir++
	} else {
		t.size = t.file.Size()
		t.nbFile++
	}
}

func (t *Tree) subtree() {
	childs, err := ioutil.ReadDir(t.path)
	if err != nil {
		return
	}
	childs = applyFilter(childs, getFilter())
	if dir_first {
		sortDirFirst(childs)
	}
	t.childs = make([]*Tree, len(childs))
	lcont := t.levelCont
	if !t.isLast() {
		l := len(lcont)
		lcont2 := make([]int, l+1)
		copy(lcont2, lcont)
		lcont2[l] = t.level
		lcont = lcont2
	}
	for i, f := range childs {
		t.childs[i] = newSubTree(t, f, i, lcont)
		t.nbDir += t.childs[i].nbDir
		t.nbFile += t.childs[i].nbFile
		t.size += t.childs[i].size
	}
}

func (t *Tree) Iterator() <-chan *Tree {
	ch := make(chan *Tree)
	go func() {
		ch <- t
		for _, c := range t.childs {
			for cc := range c.Iterator() {
				ch <- cc
			}
		}
		close(ch)
	}()
	return ch
}

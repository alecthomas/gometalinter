package lll_test

import (
	"bytes"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/walle/lll"
)

func TestShouldSkipDirs(t *testing.T) {
	skip, ret := lll.ShouldSkip(".git", true, []string{".git"}, false)
	if skip == false || ret != filepath.SkipDir {
		t.Errorf("Expected %b, %s got. %b, %s", true, filepath.SkipDir, skip, ret)
	}

	skip, ret = lll.ShouldSkip("dir", true, []string{".git"}, false)
	if skip == false || ret != nil {
		t.Errorf("Expected %b, %s got. %b, %s", true, nil, skip, ret)
	}
}

func TestShouldSkipFiles(t *testing.T) {
	skip, ret := lll.ShouldSkip("file.go", false, []string{".git"}, true)
	if skip == true || ret != nil {
		t.Errorf("Expected %b, %s got. %b, %s", false, nil, skip, ret)
	}

	skip, ret = lll.ShouldSkip("file", false, []string{".git"}, true)
	if skip == false || ret != nil {
		t.Errorf("Expected %b, %s got. %b, %s", true, nil, skip, ret)
	}

	skip, ret = lll.ShouldSkip("lll_test.go", false, []string{".git"}, false)
	if skip == true || ret != nil {
		t.Errorf("Expected %b, %s got. %b, %s", false, nil, skip, ret)
	}

	skip, ret = lll.ShouldSkip("file", false, []string{"file"}, false)
	if skip == false || ret != nil {
		t.Errorf("Expected %b, %s got. %b, %s", true, nil, skip, ret)
	}
}

func TestProcess(t *testing.T) {
	lines := "one\ntwo\ntree"
	b := bytes.NewBufferString("")
	err := lll.Process(bytes.NewBufferString(lines), b, "file", 80, 1, nil)
	if err != nil {
		t.Errorf("Expected %s, got %s", nil, err)
	}

	expected := "file:3: line is 4 characters\n"
	_ = lll.Process(bytes.NewBufferString(lines), b, "file", 3, 1, nil)
	if b.String() != expected {
		t.Errorf("Expected %s, got %s", expected, b.String())
	}
}

func TestProcessFile(t *testing.T) {
	b := bytes.NewBufferString("")
	err := lll.ProcessFile(b, "lll_test.go", 80, 1, nil)
	if err != nil {
		t.Errorf("Expected %s, got %s", nil, err)
	}
}

func TestProcessUnicode(t *testing.T) {
	lines := "日本語\n"
	b := bytes.NewBufferString("")
	expected := "file:1: line is 3 characters\n"
	_ = lll.Process(bytes.NewBufferString(lines), b, "file", 2, 1, nil)
	if b.String() != expected {
		t.Errorf("Expected %s, got %s", expected, b.String())
	}
}

func TestProcessWithTabwidth4(t *testing.T) {
	lines := "\t\t\terr := lll.ProcessFile(os.Stdout, s.Text(), " +
		"args.MaxLength, args.TabWidth, exclude)"
	b := bytes.NewBufferString("")
	expected := "file:1: line is 95 characters\n"
	_ = lll.Process(bytes.NewBufferString(lines), b, "file", 80, 4, nil)
	if b.String() != expected {
		t.Errorf("Expected %s, got %s", expected, b.String())
	}
}

func TestProcessWithTabwidth8(t *testing.T) {
	lines := "\t\t\terr := lll.ProcessFile(os.Stdout, s.Text(), " +
		"args.MaxLength, args.TabWidth, exclude)"
	b := bytes.NewBufferString("")
	expected := "file:1: line is 107 characters\n"
	_ = lll.Process(bytes.NewBufferString(lines), b, "file", 80, 8, nil)
	if b.String() != expected {
		t.Errorf("Expected %s, got %s", expected, b.String())
	}
}

func TestProcessExclude(t *testing.T) {
	lines := "one\ntwo\ntree\nTODO: fix\nFIXME: do this"
	b := bytes.NewBufferString("")
	exclude := regexp.MustCompile("TODO|FIXME")
	expected := "file:3: line is 4 characters\n"
	_ = lll.Process(bytes.NewBufferString(lines), b, "file", 3, 1, exclude)
	if b.String() != expected {
		t.Errorf("Expected %s, got %s", expected, b.String())
	}
}

func BenchmarkProcessExclude(b *testing.B) {
	lines := "one\ntwo\ntree\nTODO: fix\nFIXME: do this"
	exclude := regexp.MustCompile("TODO|FIXME")
	expected := "file:3: line is 4 characters\n"

	for i := 0; i < b.N; i++ {
		buf := bytes.NewBufferString("")
		_ = lll.Process(bytes.NewBufferString(lines), buf, "file", 3, 1, exclude)
		if buf.String() != expected {
			b.Errorf("Expected %s, got %s", expected, buf.String())
		}
	}
}

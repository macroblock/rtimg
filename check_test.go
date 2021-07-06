package main

import (
	"strings"
	"testing"

	"github.com/macroblock/imed/pkg/tagname"
	rtimg "github.com/macroblock/rtimg/pkg"
)

var (
	tableCorrect = []struct {
		skip  bool
		input string
		limit int64
	}{
		//23456789012345678901234567890
		{skip: false,
			input: "sd_2018_sobibor__12_q0w2_ar2_poster525x300.jpg",
			limit: 1000000,
		},
	}
	tableIncorrect = []string{
		//23456789012345678901234567890
		// "The_name_s01_zzz_2018__hd_q0w0",
		// "sd_2018_Sobibor__12_q0w2_trailer.mpg",
	}
)

type ttag struct {
	typ, val string
}

// TestCorrect -
func TestCorrect(t *testing.T) {
	for _, v := range tableCorrect {
		tn, err := tagname.NewFromFilename(v.input, false)
		if err != nil {
			t.Errorf("\n%q\nNewFromFilename() error:\n%v", v.input, err)
			continue
		}

		data, err := rtimg.CheckImage("", tn)
		if err != nil {
			t.Errorf("\n%q\nCheckImage() error:\n%v", v.input, err)
			continue
		}

		if data.FileSizeLimit != v.limit {
			t.Errorf("\n%q\nCheckImage() error:\n%v", v.input, err)
			continue
		}
	}
}

// TestIncorrect -
func TestIncorrect(t *testing.T) {
	for _, v := range tableIncorrect {
		tn, err := tagname.NewFromFilename(v, false)
		if err != nil {
			t.Errorf("\n%q\nNewFromFilename() error:\n%v", v, err)
			continue
		}
		sizeLimit, err := rtimg.CheckImage("", tn)
		_ = sizeLimit
		if err == nil {
			t.Errorf("\n%q\nhas no error", v)
			continue
		}
	}
}

func testEntry(t *testing.T, projectDir, projectLeaf string) {
	path := projectDir + projectLeaf
	x := strings.Split(projectDir, "/")
	projectName := x[len(x)-1]
	// key := newKey(path, "")
	key, err := rtimg.FindKey(path, nil)
	if err != nil {
		t.Errorf("TestKey: %v", err)
		return
	}
	s := key.ProjectDir()
	if s != projectDir {
		t.Errorf("TestKey incorrect project dir: " + projectDir + " != " + s)
		return
	}

	hash := key.Hash()
	if hash != "."+projectLeaf {
		t.Errorf("TestKey: invalid hash %v", hash)
	}
	name := key.Name()
	if name != projectName {
		t.Errorf("TestKey: invalid name %v", name)
	}
}

//TestKey -
func TestKey(t *testing.T) {
	projectDir := "some/path/PROJECT_NAME"
	projectLeaf := "/google_apple_feed/jpg/g_iconic_poster_600x800.jpg"
	testEntry(t, projectDir, projectLeaf)

	projectDir = "some/path/PROJECT_NAME"
	projectLeaf = "/для сервиса/600x600.jpg"
	testEntry(t, projectDir, projectLeaf)

	/*
		projectDir = "some/path/PROJECT_NAME"
		projectLeaf = "/1 сезон/600x600.jpg"
		testEntry(t, projectDir, projectLeaf)
	*/
}

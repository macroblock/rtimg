package main

import (
	"testing"

	"github.com/macroblock/imed/pkg/tagname"
	"github.com/macroblock/rtimg/pkg"
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
			limit: -1,
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

		sizeLimit, err := rtimg.CheckImage(tn, false)
		if err != nil {
			t.Errorf("\n%q\nCheckImage() error:\n%v", v.input, err)
			continue
		}

		if sizeLimit != v.limit {
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
		sizeLimit, err := rtimg.CheckImage(tn, false)
		_ = sizeLimit
		if err == nil {
			t.Errorf("\n%q\nhas no error", v)
			continue
		}
	}
}

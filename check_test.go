package main

import (
	"testing"
)

var (
	tableCorrect = []struct {
		skip bool
		input string
	}{
		//23456789012345678901234567890
		{skip: false,
			input: "sd_2018_sobibor__12_q0w2_ar2_poster525x300.jpg"},
	}
	tableIncorrect = []string{
		//23456789012345678901234567890
		"a",
		"a__",
		"2000",
		"2000__",
		"a_200",
		"a_20000",
		"_a_2000",
		"a-#_2000",
		"a_2000.trailer.ext.zzz",
		"a_2000.ext.zzz",
		"a_2000__.ext.zzz",
		"a_2000__tag__tag2",
		"a__2000",
		"The_name_s01_a_subname_2018__q0w0",
		"The_name_s01_a_subname_2018__hd_q0w0_",
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
		skip, err := checkFile(v.input, false)
		_ = skip
		if err != nil {
			t.Errorf("\n%q\ncheckFile() error:\n%v", v.input, err)
			continue
		}
	}
}

// TestIncorrect -
func TestIncorrect(t *testing.T) {
	for _, v := range tableIncorrect {
		skip, err := checkFile(v, false)
		_ = skip
		if err == nil {
			t.Errorf("\n%q\nhas no error", v)
			continue
		}
	}
}


package rtimg

import (
	"fmt"
	// "math"
	// "os"
	// "path/filepath"
	// "regexp"
	"strconv"
	"strings"
	"sync"
	"unicode/utf8"

	"github.com/malashin/ffinfo"

	ansi "github.com/malashin/go-ansi"
)

const constKilobyte = 1000

type tProps struct {
	size, ext, limit, opt string
}

type ITagname interface {
	GetTag(string) (string, error)
	Source() string
}

var mtx sync.Mutex

var rtSizes = []tProps{
	{"350x500", ".jpg", "", ""},
	{"525x300", ".jpg", "", ""},
	{"810x498", ".jpg", "", ""},
	{"270x390", ".jpg", "", ""},
	{"1620x996", ".jpg", "", ""},
	{"503x726", ".jpg", "", ""},
	{"logo", ".png", "1M", ""},
}

var gpSizes = []tProps{
	{"600x600", ".jpg", "700k", ""},
	{"600x840", ".jpg", "700k", ""},
	{"1920x1080", ".jpg", "700k", ""},
	{"1920x1080", ".jpg", "700k", "left"},
	{"1920x1080", ".jpg", "700k", "center"},
	{"1260x400", ".jpg", "700k", ""},
	{"1080x540", ".jpg", "700k", ""},
}

func atoi64(s string) (int64, error) {
	return strconv.ParseInt(s, 10, 64)
}

func parseSizeLimit(limit string) (int64, error) {
	if limit == "" {
		return -1, nil
	}
	suffix := limit[len(limit)-1]
	mult := -1
	switch suffix {
	case 'k', 'K':
		mult = constKilobyte
	case 'M':
		mult = constKilobyte * constKilobyte
	case 'G':
		mult = constKilobyte * constKilobyte * constKilobyte
	}
	if mult < 0 {
		mult = 1
	} else {
		limit = limit[:len(limit)-1]
	}
	val, err := atoi64(limit)
	return val * int64(mult), err
}

func constructNameStr(tn ITagname) (string, error) {
	ret := ""
	size, err := tn.GetTag("sizetag")
	if err != nil {
		return "", err
	}
	ret = size

	align, _ := tn.GetTag("aligntag")
	if align != "" {
		ret += " " + align
	}
	ext, _ := tn.GetTag("ext")
	if ext != "" {
		ret += " " + ext
	}
	return ret, nil
}

func constructHwStr(filePath string) (string, error) {
	probe, err := ffinfo.Probe(filePath)
	if err != nil {
		return "", err
	}
	if len(probe.Streams) < 1 {
		return "", fmt.Errorf("len(probe.Streams)<1")
	}
	codecName := strings.ToLower(probe.Streams[0].CodecName)
	switch codecName {
	default:
		codecName = "." + codecName
	case "mjpeg":
		codecName = ".jpg"
	}
	size := fmt.Sprintf("%vx%v", probe.Streams[0].Width, probe.Streams[0].Height)

	return size + " " + codecName, nil
}

func CheckImage(tn ITagname, isDeepCheck bool) (int64, error) {
	ret := int64(-1)
	filePath := tn.Source()

	typ, err := tn.GetTag("type")
	if err != nil {
		return ret, err
	}

	var list []tProps
	switch typ {
	default:
		return ret, fmt.Errorf("unsupported name format %q", typ)
	case "poster":
		list = rtSizes
	case "poster.gp":
		list = gpSizes
	}

	nameStr, err := constructNameStr(tn)
	if err != nil {
		return ret, err
	}

	if isDeepCheck {
		hwStr, err := constructHwStr(filePath)
		if err != nil {
			return ret, err
		}

		s := strings.ReplaceAll(nameStr, "left ", "")
		s = strings.ReplaceAll(s, "center ", "")
		if s != hwStr && s != "logo .png" {
			return ret, fmt.Errorf("props [%v] != file data [%v]", s, hwStr)
		}
	}

	for _, item := range list {
		s := item.size
		if item.opt != "" {
			s += " " + item.opt
		}
		s += " " + item.ext

		if s == nameStr {
			sizeLimit, err := parseSizeLimit(item.limit)
			if err != nil {
				return ret, err
			}
			return sizeLimit, nil
		}
	}

	return ret, fmt.Errorf("props [%v] is unsupported for %q", nameStr, typ)
}

func PrintColor(color int, isOk bool, filename, message string) {
	mtx.Lock()
	sign := "-"
	if isOk {
		sign = "+"
	}
	c := strconv.Itoa(color)
	ansi.Println("\x1b[" + c + ";1m" + sign + /* " " + countPad() + "/" + strconv.Itoa(length) + */ "\x1b[0m " + truncPad(filename, 50, 'r') + " \x1b[" + c + ";1m" + message + "\x1b[0m")
	mtx.Unlock()
}

func PrintYellow(filename, message string) {
	PrintColor(33, true, filename, message)
}

func PrintGreen(filename, message string) {
	PrintColor(32, true, filename, message)
}

func PrintMagenta(filename, message string) {
	PrintColor(35, true, filename, message)
}

// Pad zeroes to current file number to have the same length as overall filecount.
// func countPad() string {
// count++
// c := strconv.Itoa(count)
// pad := len(strconv.Itoa(length)) - len(c)
// for i := pad; i > 0; i-- {
// c = "0" + c
// }
// return c
// }

// truncPad truncs or pads string to needed length.
// If side is 'r' the string is padded and aligned to the right side.
// Otherwise it is aligned to the left side.
func truncPad(s string, n int, side byte) string {
	len := utf8.RuneCountInString(s)
	if len > n {
		return string([]rune(s)[0:n-3]) + "\x1b[30;1m...\x1b[0m"
	}
	if side == 'r' {
		return strings.Repeat(" ", n-len) + s
	}
	return s + strings.Repeat(" ", n-len)
}

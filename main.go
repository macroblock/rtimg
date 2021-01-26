package main

import (
	"flag"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"regexp"
	// "strconv"
	// "strings"
	"sync"
	// "unicode/utf8"

	"github.com/macroblock/rtimg/pkg"
	"github.com/macroblock/imed/pkg/tagname"
	// "github.com/malashin/ffinfo"

	ansi "github.com/malashin/go-ansi"
	"golang.org/x/crypto/ssh/terminal"
)


var mtx sync.Mutex

// TKeyVal -
type TKeyVal struct {
	key string
	val int
}

var count = 0            // Filecount for progress visualisation.
var errorsArray []string // Store errors in array.
var files []string       // Store input fileNames in global space.
var length int           // Store the amount of input files in global space.

// Flags
var threads int
var lossy bool
// var actionsFlag StringListFlags
var flagCheckOnly bool

// type tProps struct {
	// size, ext, limit, opt string
// }

// var rtSizes = []tProps{
	// // "190x230 .jpg",
	// {"350x500",   ".jpg", "", ""},
	// {"525x300",   ".jpg", "", ""},
	// // "780x100 .jpg",
	// {"810x498",   ".jpg", "", ""},
	// {"270x390",   ".jpg", "", ""},
	// {"1620x996",  ".jpg", "", ""},
	// {"503x726",   ".jpg", "", ""},
	// // "1140x726 .jpg",
	// // "3510x1089 .jpg",
	// // "100x100 .jpg",
	// // "140x140 .jpg",
	// // "1170x363 .jpg",
	// // "570x363 .jpg",
	// {"logo",      ".png", "1M",   ""},
// }

// var gpSizes = []tProps {
	// {"600x600",   ".jpg", "700k", ""},
	// {"600x840",   ".jpg", "700k", ""},
	// {"1920x1080", ".jpg", "700k", ""},
	// {"1920x1080", ".jpg", "700k", "left"},
	// {"1920x1080", ".jpg", "700k", "center"},
	// {"1260x400",  ".jpg", "700k", ""},
	// {"1080x540",  ".jpg", "700k", ""},
// }

var reErr = regexp.MustCompile(`(Error:.*)`)
var wg sync.WaitGroup
// var m sync.Mutex

// type StringListFlags []string

// func (i *StringListFlags) String() string {
    // return fmt.Sprintf("%v", *i)
// }

// func (i *StringListFlags) Set(value string) error {
    // *i = append(*i, value)
    // return nil
// }

func main() {
	// Parse input flags.
	// flag.StringVar(&format, "f", "all", "Format of the input files to compress (jpg|png|all)")
	flag.IntVar(&threads, "t", 4, "Number of threads")
	flag.BoolVar(&lossy, "l", false, "Lossy pgnquant compression for PNG files")
	// flag.BoolVar(&gazprom, "g", false, "Check Gazprom sizes instead of Rostelecom ones")
	flag.BoolVar(&flagCheckOnly, "c", false, "Check only (do not strip size)")
	// flag.IntVar(&maxsize, "m", 0, "Limit JPG output size, quality will be lowered to do this")
	// flag.StringVar(&suffixlist, "s", "", "suffix=size(:suffix=size)*")
	// flag.StringVar(&scriptFlag, "xs", nil, "Execute action string - cammand:arg{,arg};")
	// ?type:; @1; .clear; 2=0,_,1; @2; !type:; 0=1; @0; -type; $atag; +mtag:mxxx,myyy

	flag.Usage = func() {
		ansi.Println("Usage: rtimg [options] [file1 file2 ...]")
		flag.PrintDefaults()
	}
	flag.Parse()

	files = flag.Args()
	length = len(files)

	// Create channel for goroutines
	c := make(chan string)

	// Create limited number of workers.
	for i := 0; i < threads; i++ {
		wg.Add(1)
		go worker(c)
	}

	// Distribute files to free goroutines.
	for _, filePath := range files {
		c <- filePath
	}

	// Close channel.
	close(c)

	// Wait for workgroup to finish.
	wg.Wait()

	// If there were any errors.
	if len(errorsArray) > 0 {
		// Print out all the errors from the error array.
		ansi.Println("\x1b[0m\nERRORS\n========")
		for i := 0; i < len(errorsArray); i++ {
			ansi.Println(errorsArray[i])
		}
		ansi.Println("\x1b[0m========")

		// Don't close the terminal window.
		ansi.Println("Press any key to exit...")
		err := waitForAnyKey()
		if err != nil {
			ansi.Println("\x1b[31;1m"+"    [waitForAnyKey]:", err, "\x1b[0m")
		}
	}
}

func worker(c chan string) {
	defer wg.Done()
	for filePath := range c {
		fileName := filepath.Base(filePath)
		// ext := filepath.Ext(filePath)

		mtx.Lock()
		var tn rtimg.ITagname
		var err error
		tn, err = tagname.NewFromFilename(filePath, true)
		if err != nil {
			printError(fileName, err.Error())
			continue
			// return ret, err
		}
		mtx.Unlock()

		sizeLimit, err := rtimg.CheckImage(tn, true)
		if err != nil {
			printError(fileName, err.Error())
			continue
		}

		if flagCheckOnly {
			rtimg.PrintGreen(fileName, fmt.Sprintf("Ok"))
			continue
		}

		// exiftool overwrites source file if all ok
		// err = exifTool(filePath)
		// if err != nil {
			// printError(fileName, err.Error())
			// continue
		// }

		// if props.size == "" {
			// continue
		// }
		// if sizeLimit < 0 {
			// continue
		// }

		// switch props.ext {
			// default: printError(fileName, fmt.Sprintf("unsupported extension [%q] to save file", props.ext))
		// case ".jpg":
			// err = saveJPG(filePath, props)
		// case ".png":
			// err = savePNG(filePath, props)
		// }
		err = rtimg.ReduceImage(filePath, sizeLimit)
		if err != nil {
			printError(fileName, err.Error())
		}
	}
}

// func constructNameStr(tn *tagname.TTagname) (string, error) {
	// ret := ""
	// size, err := tn.GetTag("sizetag")
	// if err != nil {
		// return "", err
	// }
	// ret = size

	// align, _ := tn.GetTag("aligntag")
	// if align != "" {
		// ret += " " + align
	// }
	// ext, _ := tn.GetTag("ext")
	// if ext != "" {
		// ret += " " + ext
	// }
	// return ret, nil
// }

// func constructHwStr(filePath string) (string, error) {
	// probe, err := ffinfo.Probe(filePath)
	// if err != nil {
		// return "", err
	// }
	// if len(probe.Streams)<1 {
		// return "", fmt.Errorf("len(probe.Streams)<1")
	// }
	// codecName := strings.ToLower(probe.Streams[0].CodecName)
	// switch codecName {
	// default:
		// codecName = "." + codecName
	// case "mjpeg":
		// codecName = ".jpg"
	// }
	// size := fmt.Sprintf("%vx%v", probe.Streams[0].Width, probe.Streams[0].Height)

	// return size + " " + codecName, nil
// }

// func CheckImage(filePath string, isDeepCheck bool) (int64, error) {
	// // var resolutionString string
	// // var resolution []string
	// // fileName := filepath.Base(filePath)
	// // ret := tProps{}
	// ret := int64(-1)

	// tn, err := tagname.NewFromFilename(filePath, isDeepCheck)
	// if err != nil {
		// return ret, err
	// }
	// typ, err := tn.GetTag("type")
	// if err != nil {
		// return ret, err
	// }

	// var list []tProps
	// switch typ {
	// default:
		// return ret, fmt.Errorf("unsupported name format %q", typ)
	// case "poster":
		// list = rtSizes
	// case "poster.gp":
		// list = gpSizes
	// }

	// nameStr, err := constructNameStr(tn)
	// if err != nil {
		// return ret, err
	// }

	// if isDeepCheck {
		// hwStr, err := constructHwStr(filePath)
		// if err != nil {
			// return ret, err
		// }

		// s := strings.ReplaceAll(nameStr, "left ", "")
		// s = strings.ReplaceAll(s, "center ", "")
		// if s != hwStr && s != "logo .png" {
			// return ret, fmt.Errorf("props [%v] != file data [%v]", s, hwStr)
		// }
	// }

	// for _, item := range list {
		// s := item.size
		// if item.opt != "" {
			// s += " " + item.opt
		// }
		// s += " " + item.ext

		// if s == nameStr {
			// sizeLimit, err := parseSizeLimit(item.limit)
			// if err != nil {
				// return ret, err
			// }
			// return sizeLimit, nil
		// }
	// }

	// return ret, fmt.Errorf("props [%v] is unsupported for %q", nameStr, typ)
// }

// func atoi64(s string) (int64, error) {
	// return strconv.ParseInt(s, 10, 64)
// }

// func parseSizeLimit(limit string) (int64, error) {
	// // limit := props.limit
	// if limit == "" {
		// return -1, nil
	// }
	// suffix := limit[len(limit)-1]
	// mult := -1
	// switch suffix {
	// case 'k', 'K': mult = constKilobyte
	// case 'M': mult = constKilobyte*constKilobyte
	// case 'G': mult = constKilobyte*constKilobyte*constKilobyte
	// }
	// if mult < 0 {
		// mult = 1
	// } else {
		// limit = limit[:len(limit)-1]
	// }
	// val, err := atoi64(limit)
	// return val*int64(mult), err
// }



// round rounds floats into integer numbers.
func round(input float64) int {
	if input < 0 {
		return int(math.Ceil(input - 0.5))
	}
	return int(math.Floor(input + 0.5))
}

// waitForAnyKey await for any key press to continue.
func waitForAnyKey() error {
	fd := int(os.Stdin.Fd())
	if !terminal.IsTerminal(fd) {
		return fmt.Errorf("it's not a terminal descriptor")
	}
	state, err := terminal.MakeRaw(fd)
	if err != nil {
		return fmt.Errorf("cannot set raw mode")
	}
	defer terminal.Restore(fd, state)

	b := [1]byte{}
	os.Stdin.Read(b[:])
	return nil
}


func printError(filename, message string) {
	mtx.Lock()
	errorsArray = append(errorsArray, "\x1b[31;1m"+message+"\x1b[0m "+filename)
	rtimg.PrintColor(31, true, filename, message)
	// m.Lock()
	// ansi.Println("\x1b[31;1m- " + countPad() + "/" + strconv.Itoa(length) + "\x1b[0m " + truncPad(fileName, 50, 'r') + " \x1b[31;1m" + message + "\x1b[0m")
	// m.Unlock()
	mtx.Unlock()
}


package main

import (
	"errors"
	"flag"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"unicode/utf8"

	"github.com/malashin/ffinfo"

	ansi "github.com/malashin/go-ansi"
	"golang.org/x/crypto/ssh/terminal"
)

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
var format string
var threads int
var lossy bool
var gazprom bool
var maxsize int
var suffixlist string
var suffixes = []TKeyVal{}

var re1 = regexp.MustCompile(`^\w+_\d{4}__(?:sd|hd|3d)(?:_\w+)*_(190-230|350-500|525-300|780-100|810-498|270-390|1620-996|503-726|1140-726|3510-1089|100-100|140-140|1170-363|570-363)\.poster(?:#[0-9a-fA-F]{8})?\.(?:jpg|png)$`)

var re2 = regexp.MustCompile(`^(?:sd|hd)_\d{4}(?:_3d)?(?:_\w+)+__(?:\w+_)*poster(190x230|350x500|525x300|780x100|810x498|270x390|1620x996|503x726|1140x726|3510x1089|100x100|140x140|1170x363|570x363)(?:#[0-9a-fA-F]{8})?\.(?:jpg|png)$`)

var reGazprom = regexp.MustCompile(`^\w+_\d{4}_(?:(?:(600x600|600x840|1920x1080(?:_left|_center)?|1260x400|1080x540)\.jpg)|(?:(logo)\.png))$`)

var reErr = regexp.MustCompile(`(Error:.*)`)
var wg sync.WaitGroup
var m sync.Mutex

func parseSuffixes(in string) ([]TKeyVal, error) {
	m := map[string]bool{}
	list := []TKeyVal{}
	if strings.TrimSpace(in) == "" {
		return list, nil
	}
	for _, v := range strings.Split(in, ":") {
		x := strings.Split(v, "=")
		if len(x) != 2 {
			return nil, fmt.Errorf("while parse element %q", v)
		}
		k := strings.TrimSpace(x[0])
		s := strings.TrimSpace(x[1])
		v, err := strconv.Atoi(s)
		if err != nil {
			return nil, fmt.Errorf("%v while parse element %q", err, v)
		}
		if v < 0 {
			return nil, fmt.Errorf("negative size in element %q", v)
		}
		if _, ok := m[k]; ok {
			return nil, fmt.Errorf("duplicated key in element %q", v)
		}
		m[k] = true
		list = append(list, TKeyVal{key: k, val: v})
	}
	return list, nil
}

func main() {
	// Parse input flags.
	flag.StringVar(&format, "f", "all", "Format of the input files to compress (jpg|png|all)")
	flag.IntVar(&threads, "t", 4, "Number of threads")
	flag.BoolVar(&lossy, "l", false, "Lossy pgnquant compression for PNG files")
	flag.BoolVar(&gazprom, "g", false, "Check Gazprom sizes instead of Rostelecom ones")
	flag.IntVar(&maxsize, "m", 0, "Limit JPG output size, quality will be lowered to do this")
	flag.StringVar(&suffixlist, "s", "", "suffix=size(:suffix=size)*")
	flag.Usage = func() {
		ansi.Println("Usage: rtimg [options] [file1 file2 ...]")
		flag.PrintDefaults()
	}
	flag.Parse()

	if format != "jpg" && format != "png" && format != "all" {
		ansi.Println("\x1b[32;1mWrong --format flag, must be (jpg|png|all)\x1b[0m")
		os.Exit(1)
	}

	if maxsize < 0 {
		ansi.Println("\x1b[32;1mWrong --maxsize flag, must be >= 0\x1b[0m")
		os.Exit(1)
	}
	err := error(nil)
	suffixes, err = parseSuffixes(suffixlist)
	if err != nil {
		ansi.Println("\x1b[32;1mError: ", err, "0\x1b[0m")
		os.Exit(1)
	}

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
		ext := filepath.Ext(filePath)

		skip, err := checkFile(filePath)
		if err != nil {
			printError(fileName, err.Error())
			continue
		}
		if skip {
			continue
		}
		if ext == ".jpg" {
			saveJPG(filePath)
		}
		if ext == ".png" {
			savePNG(filePath)
		}
	}
}

func checkFile(filePath string) (bool, error) {
	var resolutionString string
	var resolution []string
	fileName := filepath.Base(filePath)
	skip := false

	// Check filenames with regexp.
	if gazprom {
		if !(reGazprom.MatchString(fileName)) {
			return skip, errors.New("WRONG FILENAME")
		}
		if reGazprom.MatchString(fileName) {
			resolutionString = reGazprom.ReplaceAllString(fileName, "${1}${2}")
			// trim possible "_left" or "_center"
			// resolutionString = strings.Split(resolutionString, "_")[0]
			fmt.Printf("resolutionString: %v\n", resolutionString)
			if resolutionString == "logo" {
				resolutionString = "1920x1080"
				skip = true
			}
			resolution = strings.Split(resolutionString, "x")
		}
	} else {
		if !(re1.MatchString(fileName) || re2.MatchString(fileName)) {
			return skip, errors.New("WRONG FILENAME")
		}
		if re1.MatchString(fileName) {
			resolutionString = re1.ReplaceAllString(fileName, "${1}")
			resolution = strings.Split(resolutionString, "-")
		}
		if re2.MatchString(fileName) {
			resolutionString = re2.ReplaceAllString(fileName, "${1}")
			resolution = strings.Split(resolutionString, "x")
		}
	}

	// Use ffprobe to check files codec and resolution.
	probe, err := ffinfo.Probe(filePath)
	if err != nil {
		return skip, err
	}
	if probe.Streams[0].CodecName != "mjpeg" && probe.Streams[0].CodecName != "png" {
		return skip, errors.New(probe.Streams[0].CodecName)
	}
	if format == "jpg" && probe.Streams[0].CodecName != "mjpeg" {
		return skip, errors.New(probe.Streams[0].CodecName)
	}
	if format == "png" && probe.Streams[0].CodecName != "png" {
		return skip, errors.New(probe.Streams[0].CodecName)
	}
	w := strconv.Itoa(probe.Streams[0].Width)
	h := strconv.Itoa(probe.Streams[0].Height)
	if (w != resolution[0]) || (h != resolution[1]) {
		return skip, errors.New(w + "x" + h)
	}
	return skip, nil
}

func getMaxSize(filename string) int {
	name := strings.TrimSuffix(filename, filepath.Ext(filename))
	for _, x := range suffixes {
		if strings.HasSuffix(name, x.key) {
			return x.val
		}
	}
	return maxsize
}

func saveJPG(filePath string) {
	fileName := filepath.Base(filePath)

	// Get input filesize.
	inputInfo, err := os.Stat(filePath)
	if err != nil {
		printError(fileName, err.Error())
		return
	}
	inputSize := round(float64(inputInfo.Size()) / 1000)

	outputSize := math.MaxInt64
	q := 0

	uptoSize := getMaxSize(fileName)

	if uptoSize > 0 {
		for outputSize > uptoSize && q <= 31 {
			// Run ffmpeg to encode file to JPEG.
			stdoutStderr, err := exec.Command("ffmpeg",
				"-i", filePath,
				"-q:v", strconv.Itoa(q),
				"-pix_fmt", "rgb24",
				"-map_metadata", "-1",
				"-loglevel", "error",
				"-y",
				filePath+"####.jpg",
			).CombinedOutput()
			if err != nil {
				printError(fileName, err.Error())
				return
			}
			if len(stdoutStderr) > 0 {
				printError(fileName, fmt.Sprintf("%v", stdoutStderr))
				return
			}

			// Get output filesize.
			outputInfo, err := os.Stat(filePath + "####.jpg")
			if err != nil {
				printError(fileName, err.Error())
				return
			}
			outputSize = round(float64(outputInfo.Size()) / 1000)
			q++
		}
	} else {
		// Run ffmpeg to encode file to JPEG.
		stdoutStderr, err := exec.Command("ffmpeg",
			"-i", filePath,
			"-q:v", strconv.Itoa(q),
			"-pix_fmt", "rgb24",
			"-map_metadata", "-1",
			"-loglevel", "error",
			"-y",
			filePath+"####.jpg",
		).CombinedOutput()
		if err != nil {
			printError(fileName, err.Error())
			return
		}
		if len(stdoutStderr) > 0 {
			printError(fileName, fmt.Sprintf("%v", stdoutStderr))
			return
		}

		// Get output filesize.
		outputInfo, err := os.Stat(filePath + "####.jpg")
		if err != nil {
			printError(fileName, err.Error())
			return
		}
		outputSize = round(float64(outputInfo.Size()) / 1000)
	}

	// Replace the original file if the size difference is higher then 1 KB.
	if (inputSize - outputSize) > 1 {
		err = os.Rename(filePath+"####.jpg", filePath)
		if err != nil {
			printError(fileName, err.Error())
			return
		}
		printGreen(fileName, strconv.Itoa(inputSize)+"KB -> "+strconv.Itoa(outputSize)+"KB")
		return
	}

	// Delete temp file if the size difference is lower then 1 KB.
	err = os.Remove(filePath + "####.jpg")
	if err != nil {
		printError(fileName, err.Error())
		return
	}
	printYellow(fileName, strconv.Itoa(inputSize)+"KB")
	return
}

func savePNG(filePath string) {
	fileName := filepath.Base(filePath)

	// Get input filesize.
	inputInfo, err := os.Stat(filePath)
	if err != nil {
		printError(fileName, err.Error())
		return
	}
	inputSize := round(float64(inputInfo.Size()) / 1000)

	if lossy {
		// Use pngquant on input file.
		err = pngQuant(filePath, filePath+"####.png")
		if err != nil {
			// Run ffmpeg to encode file to PNG.
			stdoutStderr, err := exec.Command("ffmpeg",
				"-i", filePath,
				"-q:v", "0",
				"-map_metadata", "-1",
				"-loglevel", "error",
				"-y",
				filePath+"####.png",
			).CombinedOutput()
			if len(stdoutStderr) > 0 {
				printError(fileName, fmt.Sprintf("%s", stdoutStderr))
				return
			}
			if err != nil {
				printError(fileName, err.Error())
				return
			}
			// Try using pngquant again.
			err = pngQuant(filePath+"####.png", filePath+"####.png")
			if err != nil {
				printError(fileName, err.Error())
				return
			}
		}
		return
	}
	// Use optipng on input file.
	err = optiPNG(filePath, filePath+"####.png")
	if err != nil {
		// Run ffmpeg to encode file to PNG.
		stdoutStderr, err := exec.Command("ffmpeg",
			"-i", filePath,
			"-q:v", "0",
			"-map_metadata", "-1",
			"-loglevel", "error",
			"-y",
			filePath+"####.png",
		).CombinedOutput()
		if len(stdoutStderr) > 0 {
			printError(fileName, fmt.Sprintf("%s", stdoutStderr))
			return
		}
		if err != nil {
			printError(fileName, err.Error())
			return
		}
		// Try using optipng again.
		err = optiPNG(filePath+"####.png", filePath+"####.png")
		if err != nil {
			printError(fileName, err.Error())
			return
		}
	}

	// Get output filesize.
	outputInfo, err := os.Stat(filePath + "####.png")
	if err != nil {
		printError(fileName, err.Error())
		return
	}
	outputSize := round(float64(outputInfo.Size()) / 1000)

	// Replace the original file if the size difference is higher then 1 KB.
	if (inputSize - outputSize) > 1 {
		err = os.Rename(filePath+"####.png", filePath)
		if err != nil {
			printError(fileName, err.Error())
			return
		}
		printGreen(fileName, strconv.Itoa(inputSize)+"KB -> "+strconv.Itoa(outputSize)+"KB")
		return
	}

	// Delete temp file if the size difference is lower then 1 KB.
	err = os.Remove(filePath + "####.png")
	if err != nil {
		printError(fileName, err.Error())
		return
	}
	printYellow(fileName, strconv.Itoa(inputSize)+"KB")
	return
}

// pngQuant reduces the file size of input PNG file with lossy compression.
func pngQuant(filePath string, output string) error {
	// Run pngquant to reduce the file size of input PNG file with lossy compression.
	stdoutStderr, err := exec.Command("pngquant",
		"--force",
		"--skip-if-larger",
		"--output", output,
		"--quality=0-100",
		"--speed", "1",
		"--strip",
		"--", filePath,
	).CombinedOutput()
	if len(stdoutStderr) > 0 {
		return fmt.Errorf("%s", stdoutStderr)
	}
	if err != nil {
		return err
	}
	return nil
}

// optiPNG reduces the file size of input PNG file with lossless compression.
func optiPNG(filePath string, output string) error {
	// Run pngquant to reduce the file size of input PNG file with lossy compression.
	stdoutStderr, err := exec.Command("optipng",
		"--strip", "all",
		"--out", output,
		"--", filePath,
	).CombinedOutput()
	if len(stdoutStderr) > 0 {
		if reErr.MatchString(string(stdoutStderr)) {
			return errors.New(reErr.ReplaceAllString(string(stdoutStderr), "$1"))
		}
	}
	if err != nil {
		return err
	}
	return nil
}

func printError(fileName, message string) {
	errorsArray = append(errorsArray, "\x1b[31;1m"+message+"\x1b[0m "+fileName)
	m.Lock()
	ansi.Println("\x1b[31;1m- " + countPad() + "/" + strconv.Itoa(length) + "\x1b[0m " + truncPad(fileName, 50, 'r') + " \x1b[31;1m" + message + "\x1b[0m")
	m.Unlock()
}

func printYellow(fileName, message string) {
	m.Lock()
	ansi.Println("\x1b[32;1m+ " + countPad() + "/" + strconv.Itoa(length) + "\x1b[0m " + truncPad(fileName, 50, 'r') + " \x1b[32;1m" + message + "\x1b[0m")
	m.Unlock()
}

func printGreen(fileName, message string) {
	m.Lock()
	ansi.Println("\x1b[33;1m+ " + countPad() + "/" + strconv.Itoa(length) + "\x1b[0m " + truncPad(fileName, 50, 'r') + " \x1b[33;1m" + message + "\x1b[0m")
	m.Unlock()
}

// Pad zeroes to current file number to have the same length as overall filecount.
func countPad() string {
	count++
	c := strconv.Itoa(count)
	pad := len(strconv.Itoa(length)) - len(c)
	for i := pad; i > 0; i-- {
		c = "0" + c
	}
	return c
}

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

package main

import (
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"unicode/utf8"

	"github.com/malashin/ffinfo"

	ansi "github.com/malashin/go-ansi"
	"golang.org/x/crypto/ssh/terminal"
)

var count = 0          // Filecount for progress visualisation.
var errors []string    // Store errors in array.
var args = os.Args[1:] // Convert passed arguments into array.
var length = len(args) // Store the amount of arguments in global space.
var re1 = regexp.MustCompile(`^\w+_\d{4}__(?:sd|hd|3d)(?:_\w+)*_(190-230|350-500|525-300|780-100|810-498|270-390)\.poster\.jpg$`)
var re2 = regexp.MustCompile(`^(?:sd|hd)_\d{4}(_3d)*(?:_\w+)+__(?:\w+_)*poster(190x230|350x500|525x300|780x100|810x498|270x390)\.jpg$`)
var wg sync.WaitGroup
var m sync.Mutex

func main() {
	c := make(chan string)

	// Create limited number of workers.
	for i := 0; i < runtime.NumCPU(); i++ {
		wg.Add(1)
		go worker(c)
	}

	// Distribute files to free goroutines.
	for _, filePath := range args {
		c <- filePath
	}

	// Close channel.
	close(c)

	// Wait for workgroup to finish.
	wg.Wait()

	// If there were any errors.
	if len(errors) > 0 {
		// Print out all the errors from the error array.
		ansi.Println("\x1b[0m\nERRORS\n========")
		for i := 0; i < len(errors); i++ {
			ansi.Println(errors[i])
		}
		ansi.Println("\x1b[0m========")

		// Don't close the terminal window.
		ansi.Println("Press any ket to exit...")
		err := waitForAnyKey()
		if err != nil {
			ansi.Println("\x1b[31;1m"+"    [waitForAnyKey]:", err, "\x1b[0m")
		}
	}
}

func worker(c chan string) {
	defer wg.Done()
	for filePath := range c {
		saveJpeg(filePath)
	}
}

func saveJpeg(filePath string) {
	var resolutionString string
	var resolution []string
	fileName := filepath.Base(filePath)

	// Get input filesize.
	inputInfo, err := os.Stat(filePath)
	if err != nil {
		printError(fileName, err.Error())
		return
	}
	inputSize := round(float64(inputInfo.Size()) / 1000)

	// Check filenames with regexp.
	if !(re1.MatchString(fileName) || re2.MatchString(fileName)) {
		printError(fileName, "WRONG FILENAME")
		return
	}

	// Use ffprobe to check files codec and resolution.
	probe, err := ffinfo.Probe(filePath)
	if err != nil {
		printError(fileName, err.Error())
		return
	}
	if probe.Streams[0].CodecName != "mjpeg" {
		printError(fileName, probe.Streams[0].CodecName)
		return
	}
	if re1.MatchString(fileName) {
		resolutionString = re1.ReplaceAllString(fileName, "${1}")
		resolution = strings.Split(resolutionString, "-")
	}
	if re2.MatchString(fileName) {
		resolutionString = re2.ReplaceAllString(fileName, "${1}")
		resolution = strings.Split(resolutionString, "x")
	}
	w := strconv.Itoa(probe.Streams[0].Width)
	h := strconv.Itoa(probe.Streams[0].Height)
	if (w != resolution[0]) || (h != resolution[1]) {
		printError(fileName, w+"x"+h)
		return
	}

	// Run ffmpeg to encode file to JPEG.
	stdoutStderr, err := exec.Command("ffmpeg",
		"-i", filePath,
		"-q:v", "0",
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
	outputSize := round(float64(outputInfo.Size()) / 1000)

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

func printError(fileName, message string) {
	errors = append(errors, "\x1b[31;1m"+message+"\x1b[0m "+fileName)
	m.Lock()
	ansi.Println("\x1b[31;1m- " + countPad(length) + "/" + strconv.Itoa(length) + "\x1b[0m " + truncPad(fileName, 50, 'r') + " \x1b[31;1m" + message + "\x1b[0m")
	m.Unlock()
}

func printYellow(fileName, message string) {
	m.Lock()
	ansi.Println("\x1b[32;1m+ " + countPad(length) + "/" + strconv.Itoa(length) + "\x1b[0m " + truncPad(fileName, 50, 'r') + " \x1b[32;1m" + message + "\x1b[0m")
	m.Unlock()
}

func printGreen(fileName, message string) {
	m.Lock()
	ansi.Println("\x1b[33;1m+ " + countPad(length) + "/" + strconv.Itoa(length) + "\x1b[0m " + truncPad(fileName, 50, 'r') + " \x1b[33;1m" + message + "\x1b[0m")
	m.Unlock()
}

// Pad zeroes to current file number to have the same length as overall filecount.
func countPad(length int) string {
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

package main

import (
	"flag"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sync"
	// "unicode/utf8"

	"github.com/macroblock/imed/pkg/tagname"
	"github.com/macroblock/rtimg/pkg"

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
var flagCheckOnly bool

var wg sync.WaitGroup

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
		tn, err := tagname.NewFromFilename(filePath, true)
		mtx.Unlock()
		if err != nil {
			printError(fileName, err.Error())
			continue
		}

		sizeLimit, err := rtimg.CheckImage(tn, true)
		if err != nil {
			printError(fileName, err.Error())
			continue
		}

		if flagCheckOnly {
			rtimg.PrintGreen(fileName, fmt.Sprintf("Ok"))
			continue
		}

		err = rtimg.ReduceImage(filePath, sizeLimit)
		if err != nil {
			printError(fileName, err.Error())
		}
	}
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

func printError(filename, message string) {
	mtx.Lock()
	errorsArray = append(errorsArray, "\x1b[31;1m"+message+"\x1b[0m "+filename)
	rtimg.PrintColor(31, true, filename, message)
	// m.Lock()
	// ansi.Println("\x1b[31;1m- " + countPad() + "/" + strconv.Itoa(length) + "\x1b[0m " + truncPad(fileName, 50, 'r') + " \x1b[31;1m" + message + "\x1b[0m")
	// m.Unlock()
	mtx.Unlock()
}

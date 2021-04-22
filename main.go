package main

import (
	"flag"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"unicode/utf8"

	"github.com/macroblock/imed/pkg/misc"
	"github.com/macroblock/imed/pkg/tagname"
	"github.com/macroblock/rtimg/pkg"

	ansi "github.com/malashin/go-ansi"
	"github.com/atotto/clipboard"
	"golang.org/x/crypto/ssh/terminal"
)

var mtx sync.Mutex

var count = 0            // Filecount for progress visualisation.
var errorsArray []string // Store errors in array.
var files []string       // Store input fileNames in global space.
var length int           // Store the amount of input files in global space.

// Flags
var threads int
var flagDoReduceSize bool
var flagRecursive bool
var flagNameFileRe string
var nameFileRe *regexp.Regexp

var wg sync.WaitGroup

///////////////////////////////////////////////////////////////////////////////
type RootDirData struct {
	From string
	To string
	WasErrors bool
}
var rootDirMap = map[string]RootDirData{}
var rootDirMutex = &sync.Mutex{}

func rootDirSetName(dir string, name string) error {
	err := error(nil)
	hash := strings.ReplaceAll(dir, "\\", "/")
	rootDirMutex.Lock()
	defer rootDirMutex.Unlock()
	data, ok := rootDirMap[hash]
	if ok && data.To != "" {
		err = fmt.Errorf("duplicate directory to rename it %q -> %q", dir, name)
		data.WasErrors = true
	}
	data.From = dir
	data.To = name
	rootDirMap[hash] = data
	return err
}

func rootDirSetError(dir string, err error) {
	if err == nil {
		return
	}
	hash := strings.ReplaceAll(dir, "\\", "/")
	rootDirMutex.Lock()
	defer rootDirMutex.Unlock()
	data := rootDirMap[hash]
	if data.From == "" {
		data.From = dir
	}
	data.WasErrors = true
	rootDirMap[hash] = data
}
///////////////////////////////////////////////////////////////////////////////

func main() {
	// Parse input flags.
	flag.IntVar(&threads, "t", 4, "Number of threads")
	flag.BoolVar(&flagRecursive, "d", false, "Recursive walk directories (skip symlinks)")
	flag.BoolVar(&flagDoReduceSize, "s", false, "Reduce size of the images")
	flag.StringVar(&flagNameFileRe, "n", "", "regexp that has in the first group (cannot be an empty string) a result to rename the directory")

	flag.Usage = func() {
		ansi.Println("Usage: rtimg [options] [file1 file2 ...]")
		flag.PrintDefaults()
	}
	flag.Parse()

	files = flag.Args()
	length = len(files)

	if flagNameFileRe != "" {
		nameFileRe = regexp.MustCompile(flagNameFileRe)
	}

	if clipboard.Unsupported {
		appendError("--clipboard--", fmt.Errorf("clipboard unsupported for the OS"))
	}

	if err := misc.CommandExists("ffmpeg", "exiftool", "pngquant"); err != nil {
		fmt.Printf("fatal error: %v\n", err)
		os.Exit(1)
	}

	// Create channel for goroutines
	c := make(chan string)

	// Create limited number of workers.
	for i := 0; i < threads; i++ {
		wg.Add(1)
		go worker(c)
	}

	// Distribute files to free goroutines.
	for _, filePath := range files {
		if !flagRecursive {
			c <- filePath
			continue
		}
		list, err := WalkPath(filePath)
		if err != nil {
			appendError(filePath, err)
		}
		for _, path := range list {
			c <- path
		}
	}
	close(c)
	wg.Wait()

	// rename directories
	dirlist := []RootDirData{}
	for _, v := range rootDirMap {
		dirlist = append(dirlist, v)
	}
	sort.Slice(dirlist, func(i, j int) bool {
		return len(dirlist[i].From) > len(dirlist[j].From)
	})

	for i, v := range dirlist {
		fmt.Println(i,"-", v)
	}

	errlist := []string{}
	for _, v := range dirlist {
		if v.WasErrors {
			errlist = append(errlist,
				fmt.Sprintf("%v: was errors", v.From))
			continue
		}
		if v.From == "" || v.To == "" {
			errlist = append(errlist,
				fmt.Sprintf("%v: unreachable", v.From))
		}
		err := os.Rename(v.From, v.To)
		if err != nil {
			errlist = append(errlist,
				fmt.Sprintf("%v: rename: %v", v.From, err))
		}
	}
	errorsArray = append(errorsArray, errlist...)

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

func WalkPath(path string) ([]string, error) {
	ret := []string{}
	err := filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		// check if it is the specified filename that contains a name to rename the directory
		if nameFileRe != nil {
			filename := filepath.Base(path)
			dir := filepath.Dir(path)
			val := nameFileRe.FindAllStringSubmatch(filename, -1)
			if val != nil && len(val) == 1 && len(val[0]) == 2 && val[0][1] != "" {
				err := rootDirSetName(dir, val[0][1])
				if err != nil {
					setError(dir, err)
				}
				return nil
			}
			if val != nil {
				setError(path, fmt.Errorf("incorrect result of regexp %q", flagNameFileRe))
				return nil
			}
		}
		ret = append(ret, path)
		return nil
	})

	return ret, err
}

func worker(c chan string) {
	defer wg.Done()
	for filePath := range c {
		workerProcess(filePath)
	}
}

func workerProcess(filePath string) {
	fileNamePath := filePath
	fileName := filepath.Base(filePath)
	filePath, err := filepath.Abs(filePath)
	if err != nil {
		setError(fileNamePath, err)
		return
	}

	mtx.Lock()
	// !!!TODO!!! something with deep check
	tn, err := tagname.NewFromFilename(filePath, true)
	mtx.Unlock()
	if err != nil {
		tn = nil
	}

	data, err := rtimg.CheckImage(filePath, tn)
	if err != nil {
		setError(fileNamePath, err)
		return
	}

	sizeLimit := data.FileSizeLimit
	if sizeLimit < 0 {
		// RenameRootDir(filePath)
		printGreen(fileName, "Ok")
		return
	}

	inputSize, err := rtimg.GetFileSize(filePath)
	if err != nil {
		setError(fileNamePath, err)
		return
	}

	if !flagDoReduceSize {
		if inputSize > sizeLimit {
			setError(fileNamePath, fmt.Errorf("%v KB > %v KB", inputSize/1000, sizeLimit/1000))
		} else {
			printGreen(fileName, "Ok")
		}
		return
	}

	outputSize, q, err := rtimg.ReduceImage(filePath, data.FileSizeLimit)
	if err != nil {
		setError(fileNamePath, err)
		return
	}
	if inputSize == outputSize {
		printGreen(fileName, "Ok")
		return
	}
	msg := fmt.Sprintf("%v KB < %v KB, q: %v d: %v", outputSize/1000, sizeLimit/1000, q, inputSize-outputSize)
	if q > 13 { // !!!FIXME: empirical value
		printMagenta(fileName, msg)
	} else {
		printYellow(fileName, msg)
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

func printColor(color int, isOk bool, filename, message string) {
	mtx.Lock()
	sign := "-"
	if isOk {
		sign = "+"
	}
	c := strconv.Itoa(color)
	ansi.Println("\x1b[" + c + ";1m" + sign + " " + countPad() + "/" + strconv.Itoa(length) +  "\x1b[0m " + truncPad(filename, 50, 'r') + " \x1b[" + c + ";1m" + message + "\x1b[0m")
	mtx.Unlock()
}

func hasErrors() bool {
	return len(errorsArray) > 0
}

func appendError(filename string, err error) {
	if err == nil {
		return
	}
	errorsArray = append(errorsArray, "\x1b[31;1m" + err.Error() + "\x1b[0m " +
		filepath.Base(filename) + " ->\x1b[35m" + filepath.Dir(filename))
}

func setError(path string, err error) {
	projectDir := rtimg.GetProjectDir(path)
	if projectDir == "" {
		projectDir = filepath.Dir(path)
	}
	rootDirSetError(projectDir, err)
	appendError(path, err)
	printColor(31, false, filepath.Base(path), "#"+projectDir+"#"+err.Error())
}

func printYellow(filename, message string) {
	printColor(33, true, filename, message)
}

func printGreen(filename, message string) {
	printColor(32, true, filename, message)
}

func printMagenta(filename, message string) {
	printColor(35, true, filename, message)
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

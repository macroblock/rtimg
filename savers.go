package main

import (
	// "errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

func getFileSize(filename string) (int64, error) {
	info, err := os.Stat(filename)
	if err != nil {
		return -1, err
	}
	return info.Size(), nil
}

func reduceJPG(nameIn, nameOut string, limitSize int64) (int64, int, error) {
	q := 0
	outputSize := int64(-1)
	for q <= 31 {
		// fmt.Println("q-->",q)
		// Run ffmpeg to encode file to JPEG.
		stdoutStderr, err := exec.Command("ffmpeg",
			"-i", nameIn,
			"-q:v", fmt.Sprintf("%v", q),
			"-pix_fmt", "rgb24",
			"-map_metadata", "-1",
			"-loglevel", "error",
			"-y",
			nameOut,
		).CombinedOutput()
		if err != nil {
			return -1, -1, err
		}
		if len(stdoutStderr) > 0 {
			return -1, -1, fmt.Errorf("%v", stdoutStderr)
		}

		outputSize, err = getFileSize(nameOut)
		if err != nil {
			return -1, -1, err
		}
		if outputSize <= limitSize {
			break
		}
		q++
	}

	if outputSize > limitSize {
		return -1, -1, fmt.Errorf("cannot reduce file size (%v>%v)", outputSize, limitSize)
	}
	return outputSize, q, nil
}


func reducePNG(nameIn, nameOut string, limitSize int64) (int64, int, error) {
	err := pngQuant(nameIn, nameOut)
	if err != nil {
		// Run ffmpeg to encode file to PNG.
		stdoutStderr, err := exec.Command("ffmpeg",
			"-i", nameIn,
			"-q:v", "0",
			"-map_metadata", "-1",
			"-loglevel", "error",
			"-y",
			nameOut,
		).CombinedOutput()
		if len(stdoutStderr) > 0 {
			return -1, -1, fmt.Errorf("%v", stdoutStderr)
		}
		if err != nil {
			return -1, -1, err
		}
		// Try using pngquant again.
		err = pngQuant(nameOut, nameOut)
		if err != nil {
			return -1, -1, err
		}
	}
	outputSize, err := getFileSize(nameOut)
	if err != nil {
		return -1, -1, err
	}
	if outputSize > limitSize {
		return -1, -1, fmt.Errorf("cannot reduce file size (%v>%v)", outputSize, limitSize)
	}
	return outputSize, -1, nil
}


func reduceImageFile(filePath string, props tProps) error {
	fileName := filepath.Base(filePath)

	inputSize, err := getFileSize(filePath)
	if err != nil {
		return err
	}

	limitSize, err := getMaxSize(props)
	if err != nil {
		return err
	}

	if inputSize <= limitSize || limitSize < 0 {
		printGreen(fileName, fmt.Sprintf("%v <= %v", inputSize/1000, limitSize/1000))
		return nil
	}


	// for q <= 31 {
		// // fmt.Println("q-->",q)
		// // Run ffmpeg to encode file to JPEG.
		// stdoutStderr, err := exec.Command("ffmpeg",
			// "-i", filePath,
			// "-q:v", fmt.Sprintf("%v", q),
			// "-pix_fmt", "rgb24",
			// "-map_metadata", "-1",
			// "-loglevel", "error",
			// "-y",
			// filePath+"####.jpg",
		// ).CombinedOutput()
		// if err != nil {
			// return err
		// }
		// if len(stdoutStderr) > 0 {
			// return fmt.Errorf("%v", stdoutStderr)
		// }

		// outputSize, err := getFileSize(filePath + "####.jpg")
		// if err != nil {
			// return err
		// }
		// if outputSize <= limitSize {
			// break
		// }
		// q++
	// }

	// if outputSize > limitSize {
		// os.Remove(filePath + "####.jpg")
		// return fmt.Errorf("cannot reduce file size (%v>%v)", outputSize, limitSize)
	// }

	nameIn := filePath
	nameOut := ""
	outputSize := int64(-1)
	q := -1

	switch props.ext {
	default: printError(fileName, fmt.Sprintf("unsupported extension [%q] to process file", props.ext))
	case ".jpg":
		nameOut = filePath + "####.jpg"
		outputSize, q, err = reduceJPG(nameIn, nameOut, limitSize)
	case ".png":
		nameOut = filePath + "####.png"
		outputSize, q, err = reducePNG(nameIn, nameOut, limitSize)
		// err = savePNG(filePath, props)
	}
	if err != nil {
		// !!!FIXME: it's not good behavior to skip error checks
		err := os.Remove(nameOut)
		_ = err
		return err
	}

	err = os.Rename(nameOut, nameIn)
	if err != nil {
		return err
	}

	msg := fmt.Sprintf("%vKB -> %vKB, q%v", inputSize/1000, outputSize/1000, q)
	// q < -1 for .png
	if q > 13 || q < 0 { // !!!FIXME: empirical value
		printMagenta(fileName, msg)
	} else {
		printYellow(fileName, msg)
	}
	return nil
}

func savePNG(filePath string, props tProps) error {
	fileName := filepath.Base(filePath)

	// Get input filesize.
	inputInfo, err := os.Stat(filePath)
	if err != nil {
		return err
	}
	inputSize := inputInfo.Size()
	limitSize, err := getMaxSize(props)
	if err != nil {
		return err
	}

	if inputSize <= limitSize || limitSize < 0 {
		printGreen(fileName, fmt.Sprintf("%v <= %v", inputSize/1000, limitSize/1000))
		return nil
	}

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
			// printError(fileName, fmt.Sprintf("%s", stdoutStderr))
			return fmt.Errorf("%v", stdoutStderr)
		}
		if err != nil {
			return err
		}
		// Try using pngquant again.
		err = pngQuant(filePath+"####.png", filePath+"####.png")
		if err != nil {
			// printError(fileName, err.Error())
			return err
		}
	}

	// Get output filesize.
	outputInfo, err := os.Stat(filePath + "####.png")
	if err != nil {
		// printError(fileName, err.Error())
		return err
	}
	// outputSize := round(float64(outputInfo.Size()) / 1000)
	outputSize := outputInfo.Size()

	// Replace the original file if the size difference is higher then 1 KB.
	if (inputSize - outputSize) > 1000 {
		err = os.Rename(filePath+"####.png", filePath)
		if err != nil {
			// printError(fileName, err.Error())
			return err
		}
		// printGreen(fileName, strconv.Itoa(inputSize/1000)+"KB -> "+strconv.Itoa(outputSize/1000)+"KB")
		printGreen(fileName, fmt.Sprintf("%vKB -> %vKB", inputSize/1000, outputSize/1000))
		return nil
	}

	// Delete temp file if the size difference is lower then 1 KB.
	err = os.Remove(filePath + "####.png")
	if err != nil {
		// printError(fileName, err.Error())
		return err
	}
	printYellow(fileName, fmt.Sprintf("%vKB",inputSize/1000))
	return nil
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

// // optiPNG reduces the file size of input PNG file with lossless compression.
// func optiPNG(filePath string, output string) error {
	// // Run pngquant to reduce the file size of input PNG file with lossy compression.
	// stdoutStderr, err := exec.Command("optipng",
		// "--strip", "all",
		// "--out", output,
		// "--", filePath,
	// ).CombinedOutput()
	// if len(stdoutStderr) > 0 {
		// if reErr.MatchString(string(stdoutStderr)) {
			// return errors.New(reErr.ReplaceAllString(string(stdoutStderr), "$1"))
		// }
	// }
	// if err != nil {
		// return err
	// }
	// return nil
// }

func exifTool(filePath string) error {
	// Run pngquant to reduce the file size of input PNG file with lossy compression.
	stdoutStderr, err := exec.Command("exiftool",
		"-overwrite_original",
		"-all=", filePath,
	).CombinedOutput()
	// if len(stdoutStderr) > 0 {
		// return fmt.Errorf("%s", stdoutStderr)
	// }
	if err != nil {
		// return err
		return fmt.Errorf("error: %s\ndata:\n%v", err.Error(), stdoutStderr)
	}
	return nil
}

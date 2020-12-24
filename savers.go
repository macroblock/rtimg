package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

func saveJPG(filePath string, props tProps) error {
	fileName := filepath.Base(filePath)

	// Get input filesize.
	inputInfo, err := os.Stat(filePath)
	if err != nil {
		// printError(fileName, err.Error())
		return err
	}
	// inputSize := round(float64(inputInfo.Size() / 1000)
	inputSize := inputInfo.Size()

	// outputSize := int64(math.MaxInt64)
	outputSize := int64(-1)
	q := 0

	// uptoSize := getMaxSize(fileName)
	limitSize, err := maxSize(props)
	if err != nil {
		return err
	}

	if inputSize <= limitSize || limitSize < 0 {
		printGreen(fileName, fmt.Sprintf("%v <= %v", inputSize/1000, limitSize/1000))
		return nil
	}

	for q <= 31 {
		// fmt.Println("q-->",q)
		// Run ffmpeg to encode file to JPEG.
		stdoutStderr, err := exec.Command("ffmpeg",
			"-i", filePath,
			"-q:v", fmt.Sprintf("%v", q),
			"-pix_fmt", "rgb24",
			"-map_metadata", "-1",
			"-loglevel", "error",
			"-y",
			filePath+"####.jpg",
		).CombinedOutput()
		if err != nil {
			// printError(fileName, err.Error())
			return err
		}
		if len(stdoutStderr) > 0 {
			// printError(fileName, fmt.Sprintf("%v", stdoutStderr))
			return fmt.Errorf("%v", stdoutStderr)
		}

		// Get output filesize.
		outputInfo, err := os.Stat(filePath + "####.jpg")
		if err != nil {
			// printError(fileName, err.Error())
			return err
		}
		// outputSize = round(float64(outputInfo.Size()) / 1000)
		outputSize = outputInfo.Size()
		// size, err := saveJPGfn(filePath, filePath+"####.jpg", q)
		if err != nil {
			return err
		}
		if outputSize <= limitSize {
			break
		}
		q++
	}

	// Replace the original file if the size difference is higher then 1 KB.
	if (inputSize - outputSize) > 1000 {
		err = os.Rename(filePath+"####.jpg", filePath)
		if err != nil {
			// printError(fileName, err.Error())
			return err
		}
		printGreen(fileName, fmt.Sprintf("%vKB -> %vKB, q%v", inputSize/1000, outputSize/1000, q))
		return nil
	}

	// Delete temp file if the size difference is lower then 1 KB.
	err = os.Remove(filePath + "####.jpg")
	if err != nil {
		// printError(fileName, err.Error())
		return nil
	}
	printYellow(fileName, fmt.Sprintf("%vKB, q%v", inputSize/1000, q))
	return nil
}

func savePNG(filePath string, props tProps) error {
	fileName := filepath.Base(filePath)

	// Get input filesize.
	inputInfo, err := os.Stat(filePath)
	if err != nil {
		// printError(fileName, err.Error())
		return err
	}
	// inputSize := round(float64(inputInfo.Size()) / 1000)
	inputSize := inputInfo.Size()

	limitSize, err := maxSize(props)
	if err != nil {
		return err
	}

	if inputSize <= limitSize || limitSize < 0 {
		printGreen(fileName, fmt.Sprintf("%v <= %v", inputSize/1000, limitSize/1000))
		return nil
	}

	// if lossy {
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
				// printError(fileName, fmt.Sprintf("%s", stdoutStderr))
				return fmt.Errorf("%v", stdoutStderr)
			}
			if err != nil {
				// printError(fileName, err.Error())
				return err
			}
			// Try using pngquant again.
			err = pngQuant(filePath+"####.png", filePath+"####.png")
			if err != nil {
				// printError(fileName, err.Error())
				return err
			}
		}
		// return nil
	// }

	// // Use optipng on input file.
	// err = optiPNG(filePath, filePath+"####.png")
	// if err != nil {
		// // Run ffmpeg to encode file to PNG.
		// stdoutStderr, err := exec.Command("ffmpeg",
			// "-i", filePath,
			// "-q:v", "0",
			// "-map_metadata", "-1",
			// "-loglevel", "error",
			// "-y",
			// filePath+"####.png",
		// ).CombinedOutput()
		// if len(stdoutStderr) > 0 {
			// // printError(fileName, fmt.Sprintf("%s", stdoutStderr))
			// return fmt.Errorf("%v", stdoutStderr)
		// }
		// if err != nil {
			// // printError(fileName, err.Error())
			// return err
		// }
		// // Try using optipng again.
		// err = optiPNG(filePath+"####.png", filePath+"####.png")
		// if err != nil {
			// // printError(fileName, err.Error())
			// return err
		// }
	// }

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

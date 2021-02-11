package rtimg

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func GetFileSize(filename string) (int64, error) {
	info, err := os.Stat(filename)
	if err != nil {
		return -1, err
	}
	return info.Size(), nil
}

func ReduceJPG(nameIn, nameOut string, limitSize int64) (int64, int, error) {
	q := 0
	outputSize := int64(-1)
	for q <= 31 {
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

		outputSize, err = GetFileSize(nameOut)
		if err != nil {
			return -1, -1, err
		}
		if outputSize <= limitSize {
			return outputSize, q, nil
		}
		q++
	}
	return -1, -1, fmt.Errorf("cannot reduce file size (%v>%v)", outputSize, limitSize)
}

func ReducePNG(nameIn, nameOut string, limitSize int64) (int64, int, error) {
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
	outputSize, err := GetFileSize(nameOut)
	if err != nil {
		return -1, -1, err
	}
	if outputSize > limitSize {
		return -1, -1, fmt.Errorf("cannot reduce file size (%v>%v)", outputSize, limitSize)
	}
	return outputSize, -1, nil
}

// ReduceImage - returns outputSize, q, error
func ReduceImage(filePath string, sizeLimit int64) (int64, int, error) {
	err := exifTool(filePath)
	if err != nil {
		return -1, -1, err
	}

	inputSize, err := GetFileSize(filePath)
	if err != nil {
		return -1, -1, err
	}

	if inputSize <= sizeLimit || sizeLimit < 0 {
		// PrintGreen(fileName, "Ok")
		return inputSize, -1, nil
	}

	nameIn := filePath
	nameOut := ""
	outputSize := int64(-1)
	q := -1

	ext := strings.ToLower(filepath.Ext(nameIn))
	switch ext {
	default:
		return -1, -1, fmt.Errorf("unsupported extension [%q] to process file", ext)
	case ".jpg":
		nameOut = filePath + "####.jpg"
		outputSize, q, err = ReduceJPG(nameIn, nameOut, sizeLimit)
	case ".png":
		nameOut = filePath + "####.png"
		outputSize, q, err = ReducePNG(nameIn, nameOut, sizeLimit)
	}
	if err != nil {
		// !!!FIXME: it's not good behavior to skip error checks
		err := os.Remove(nameOut)
		_ = err
		return -1, -1, err
	}

	err = os.Rename(nameOut, nameIn)
	if err != nil {
		return -1, -1, err
	}

	return outputSize, q, nil
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
		return fmt.Errorf("%q", string(stdoutStderr))
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
	if err != nil {
		// return err
		return fmt.Errorf("error: %s\ndata:\n%q", err.Error(), string(stdoutStderr))
	}
	return nil
}

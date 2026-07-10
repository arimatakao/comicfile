// Package container provides shared implementation details for output formats.
package container

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ErrPageIndexOutOfRange reports an attempt to read a page outside the
// container's page range.
var ErrPageIndexOutOfRange = errors.New("page index out of range")

func SafeOutputPath(outputDir, outputFileName, extension string) string {
	outputFileName = SafeOutputName(outputFileName)
	outputPath := filepath.Join(outputDir, outputFileName+"."+extension)
	for count := 1; ; count++ {
		if _, err := os.Stat(outputPath); errors.Is(err, os.ErrNotExist) {
			return outputPath
		}
		outputPath = filepath.Join(outputDir, fmt.Sprintf("%s (%d).%s", outputFileName, count, extension))
	}
}

func SafeOutputDirPath(outputDir, outputFileName string) string {
	outputFileName = SafeOutputName(outputFileName)
	outputPath := filepath.Join(outputDir, outputFileName)
	for count := 1; ; count++ {
		if _, err := os.Stat(outputPath); errors.Is(err, os.ErrNotExist) {
			return outputPath
		}
		outputPath = filepath.Join(outputDir, fmt.Sprintf("%s (%d)", outputFileName, count))
	}
}

func SafeOutputName(outputFileName string) string {
	for _, character := range []string{"/", `\\`, "<", ">", ":", `"`, "?", "*"} {
		outputFileName = strings.ReplaceAll(outputFileName, character, "_")
	}
	outputFileName = strings.ReplaceAll(outputFileName, "|", "-")
	if outputFileName == "" || outputFileName == "." || outputFileName == ".." {
		return "_"
	}
	return outputFileName
}

package comicfile

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/arimatakao/comicfile/metadata"
)

const (
	// CBZ_EXT is a CBZ archive container extension.
	CBZ_EXT = "cbz"
	// PDF_EXT is a PDF container extension.
	PDF_EXT = "pdf"
	// EPUB_EXT is an EPUB container extension.
	EPUB_EXT = "epub"
	// DIR_EXT stores chapter pages in a plain directory.
	DIR_EXT = "dir"
)

// ErrExtensionNotSupport is returned when a requested output container
// extension is unknown.
var ErrExtensionNotSupport = errors.New("extension container is not supported")

// IsNotSupported reports whether fileFormat is not one of the supported
// container extensions.
func IsNotSupported(fileFormat string) bool {
	return fileFormat != CBZ_EXT &&
		fileFormat != PDF_EXT &&
		fileFormat != EPUB_EXT &&
		fileFormat != DIR_EXT
}

// Container describes a writable chapter output that accepts page images and
// then persists itself to disk.
type ContainerWriter interface {
	// WriteOnDiskAndClose finalizes container content and writes it into
	// outputDir using outputFileName as a base name.
	WriteOnDiskAndClose(outputDir string, outputFileName string, m metadata.Metadata, chapterRange string) error
	// AddPage appends a new page represented by imageBytes with fileExt format.
	AddPage(fileExt string, imageBytes []byte) error
}

// NewContainer creates a container by file extension.
//
// Supported extensions are CBZ_EXT, PDF_EXT, EPUB_EXT and DIR_EXT.
func NewContainer(extension string) (ContainerWriter, error) {

	switch extension {
	case CBZ_EXT:
		return newCBZArchive()
	case PDF_EXT:
		return newPdfFile()
	case EPUB_EXT:
		return newEpubArchive()
	case DIR_EXT:
		return newDirContainer()
	}

	return nil, ErrExtensionNotSupport
}

// ContainerReader describes a readable comic chapter container.
type ContainerReader interface {
	// TotalPages returns the number of pages in the container.
	TotalPages() int
	// ErrPages returns the number of pages that could not be read.
	ErrPages() int
	// Page returns the image data for the page at index.
	Page(index int) ([]byte, error)
}

// func OpenContainer(filepath string) (ContainerReader, err)

// safeOutputPath returns a non-existing output path for a file container.
// It sanitizes outputFileName and appends a numeric suffix when needed to
// avoid overwriting an existing file.
func safeOutputPath(outputDir, outputFileName, extension string) string {
	outputFileName = SafeOutputName(outputFileName)

	outputPath := filepath.Join(outputDir, outputFileName+"."+extension)

	for count := 1; ; count++ {
		_, err := os.Stat(outputPath)
		if errors.Is(err, os.ErrNotExist) {
			break
		}
		outputPath = filepath.Join(outputDir,
			fmt.Sprintf("%s (%d).%s", outputFileName, count, extension))
	}
	return outputPath
}

// safeOutputDirPath returns a non-existing output path for a directory
// container. It sanitizes outputFileName and appends a numeric suffix when
// the target directory already exists.
func safeOutputDirPath(outputDir, outputFileName string) string {
	outputFileName = SafeOutputName(outputFileName)

	outputPath := filepath.Join(outputDir, outputFileName)

	for count := 1; ; count++ {
		_, err := os.Stat(outputPath)
		if errors.Is(err, os.ErrNotExist) {
			break
		}
		outputPath = filepath.Join(outputDir,
			fmt.Sprintf("%s (%d)", outputFileName, count))
	}
	return outputPath
}

// SafeOutputName replaces path separators and characters that are invalid in
// common output file systems while preserving a single file or directory name.
func SafeOutputName(outputFileName string) string {
	// unix
	outputFileName = strings.ReplaceAll(outputFileName, "/", "_")
	outputFileName = strings.ReplaceAll(outputFileName, `\`, "_")
	// windows
	outputFileName = strings.ReplaceAll(outputFileName, "<", "_")
	outputFileName = strings.ReplaceAll(outputFileName, ">", "_")
	outputFileName = strings.ReplaceAll(outputFileName, ":", "_")
	outputFileName = strings.ReplaceAll(outputFileName, `"`, "_")
	outputFileName = strings.ReplaceAll(outputFileName, "?", "_")
	outputFileName = strings.ReplaceAll(outputFileName, "*", "_")
	outputFileName = strings.ReplaceAll(outputFileName, "|", "-")
	if outputFileName == "" || outputFileName == "." || outputFileName == ".." {
		return "_"
	}
	return outputFileName
}

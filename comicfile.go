package comicfile

import (
	"image"
	"os"
	"path/filepath"
	"strings"

	"github.com/arimatakao/comicfile/cbz"
	"github.com/arimatakao/comicfile/dir"
	"github.com/arimatakao/comicfile/epub"
	"github.com/arimatakao/comicfile/internal/container"
	"github.com/arimatakao/comicfile/metadata"
	"github.com/arimatakao/comicfile/pdf"
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
		return cbz.New()
	case PDF_EXT:
		return pdf.New()
	case EPUB_EXT:
		return epub.New()
	case DIR_EXT:
		return dir.New()
	}

	return nil, container.ErrExtensionNotSupported
}

// ContainerReader describes a readable comic chapter container.
type ContainerReader interface {
	// TotalPages returns the number of pages in the container.
	TotalPages() int
	// ErrPages returns the number of pages that could not be read.
	ErrPages() int
	// Metadata returns the metadata stored in the container.
	Metadata() *metadata.Metadata
	// Page returns the image for the page at index.
	Page(index int) (image.Image, error)
}

// OpenContainer opens a readable comic chapter container.
//
// Directory, CBZ, EPUB, and PDF containers are supported.
func OpenContainer(path string) (ContainerReader, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}

	if info.IsDir() {
		return dir.Open(path)
	}
	if strings.EqualFold(filepath.Ext(path), "."+CBZ_EXT) {
		return cbz.Open(path)
	}
	if strings.EqualFold(filepath.Ext(path), "."+EPUB_EXT) {
		return epub.Open(path)
	}
	if strings.EqualFold(filepath.Ext(path), "."+PDF_EXT) {
		return pdf.Open(path)
	}

	return nil, container.ErrExtensionNotSupported
}

// SafeOutputName replaces path separators and characters that are invalid in
// common output file systems while preserving a single file or directory name.
func SafeOutputName(outputFileName string) string {
	return container.SafeOutputName(outputFileName)
}

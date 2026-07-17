package comicfile

import (
	"image"
	"io"
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
	// CBZ_EXT is the filename extension for CBZ archives.
	CBZ_EXT = "cbz"
	// PDF_EXT is the filename extension for PDF documents.
	PDF_EXT = "pdf"
	// EPUB_EXT is the filename extension for EPUB books.
	EPUB_EXT = "epub"
	// DIR_EXT identifies the directory-based output format.
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

// ContainerWriter accepts comic page images and writes a completed chapter to
// disk.
type ContainerWriter interface {
	// WriteOnDiskAndClose finalizes the container and writes it below outputDir.
	// outputFileName is used as the output name; m supplies metadata, and
	// chapterRange optionally replaces the chapter portion of an EPUB title.
	WriteOnDiskAndClose(outputDir string, outputFileName string, m metadata.Metadata, chapterRange string) error
	// AddPage appends imageBytes as a page. fileExt is the image filename
	// extension, without a leading period.
	AddPage(fileExt string, imageBytes []byte) error
}

// NewContainer creates a container by file extension.
//
// extension must be one of CBZ_EXT, PDF_EXT, EPUB_EXT, or DIR_EXT.
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

// ContainerReader provides page images and metadata from a comic chapter
// container.
type ContainerReader interface {
	// Close releases resources held by the reader.
	io.Closer
	// TotalPages returns the number of pages in the container.
	TotalPages() int
	// ErrPages returns the number of pages that could not be read.
	ErrPages() int
	// Metadata returns the metadata stored in the container.
	Metadata() *metadata.Metadata
	// Page returns the image for the page at index.
	Page(index int) (image.Image, error)
}

// OpenContainer opens the comic chapter container at path.
//
// The container format is selected from the path: a directory, or a file with
// a CBZ_EXT, EPUB_EXT, or PDF_EXT extension.
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

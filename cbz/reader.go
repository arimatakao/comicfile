package cbz

import (
	"archive/zip"
	"encoding/json"
	"encoding/xml"
	"image"

	"github.com/arimatakao/comicfile/internal/container"
	"github.com/arimatakao/comicfile/metadata"
)

type cbzReader struct {
	pages    []image.Image
	errPages int
	metadata metadata.Metadata
}

// Open creates a reader for the image files in a CBZ archive.
func Open(path string) (*cbzReader, error) {
	archive, err := zip.OpenReader(path)
	if err != nil {
		return nil, err
	}
	defer archive.Close()

	reader := &cbzReader{pages: make([]image.Image, 0, len(archive.File))}
	if archive.Comment != "" {
		_ = json.Unmarshal([]byte(archive.Comment), &reader.metadata.CBI)
	}
	for _, file := range archive.File {
		if file.FileInfo().IsDir() {
			continue
		}
		if file.Name == "ComicInfo.xml" {
			_ = readComicInfo(file, &reader.metadata.CI)
			continue
		}

		page, err := readCBZImage(file)
		if err != nil {
			reader.errPages++
			continue
		}
		reader.pages = append(reader.pages, page)
	}

	return reader, nil
}

// TotalPages returns the number of readable images in the CBZ archive.
func (c *cbzReader) TotalPages() int {
	return len(c.pages)
}

// ErrPages returns the number of archive entries that could not be read.
func (c *cbzReader) ErrPages() int {
	return c.errPages
}

// Metadata returns the metadata stored in the CBZ archive.
func (c *cbzReader) Metadata() *metadata.Metadata {
	return &c.metadata
}

// Page returns the decoded image at index.
func (c *cbzReader) Page(index int) (image.Image, error) {
	if index < 0 || index >= len(c.pages) {
		return nil, container.ErrPageIndexOutOfRange
	}

	return c.pages[index], nil
}

func readCBZImage(file *zip.File) (image.Image, error) {
	reader, err := file.Open()
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	page, _, err := image.Decode(reader)
	return page, err
}

func readComicInfo(file *zip.File, comicInfo *metadata.ComicInfoMetadata) error {
	reader, err := file.Open()
	if err != nil {
		return err
	}
	defer reader.Close()

	return xml.NewDecoder(reader).Decode(comicInfo)
}

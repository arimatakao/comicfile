package cbz

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"image"
	"io"
	"os"

	"github.com/arimatakao/comicfile/internal/container"
	"github.com/arimatakao/comicfile/metadata"
)

type cbzArchive struct {
	buf         *bytes.Buffer
	writer      *zip.Writer
	pageCounter int
}

type cbzReader struct {
	pages    []image.Image
	errPages int
	metadata metadata.Metadata
}

// New creates an in-memory CBZ archive ready to receive pages.
func New() (*cbzArchive, error) {
	buf := new(bytes.Buffer)

	zipWriter := zip.NewWriter(buf)

	c := cbzArchive{
		buf:         buf,
		writer:      zipWriter,
		pageCounter: 1,
	}

	return &c, nil
}

// WriteOnDiskAndClose adds ComicBookInfo and ComicInfo metadata, closes the
// archive, and writes it to a unique CBZ file in outputDir.
func (c *cbzArchive) WriteOnDiskAndClose(outputDir, outputFileName string,
	m metadata.Metadata, chapterRange string) error {
	if err := os.MkdirAll(outputDir, os.ModePerm); err != nil {
		return err
	}

	// ComicBookInfo metadata
	comment, err := json.Marshal(m.CBI)
	if err != nil {
		return err
	}
	err = c.writer.SetComment(string(comment))
	if err != nil {
		return err
	}

	// ComicRack metadata
	w, err := c.writer.Create("ComicInfo.xml")
	if err != nil {
		return err
	}

	comicInfoContent, err := xml.Marshal(m.CI)
	if err != nil {
		return err
	}

	cireader := bytes.NewReader(comicInfoContent)
	if _, err := io.Copy(w, cireader); err != nil {
		return err
	}

	outputPath := container.SafeOutputPath(outputDir, outputFileName, "cbz")

	err = c.writer.Close()
	if err != nil {
		return err
	}

	err = os.WriteFile(outputPath, c.buf.Bytes(), os.ModePerm)
	if err != nil {
		return err
	}

	return nil
}

// AddPage stores image bytes as the next zero-padded page file in the archive.
func (c *cbzArchive) AddPage(fileExt string, src []byte) error {
	fileName := fmt.Sprintf("%02d.%s", c.pageCounter, fileExt)
	buf := bytes.NewBuffer(src)
	w, err := c.writer.Create(fileName)
	if err != nil {
		return err
	}
	if _, err := io.Copy(w, buf); err != nil {
		return err
	}
	c.pageCounter++
	return nil
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

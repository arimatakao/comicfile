package comicfile

import (
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"os"
	"path/filepath"

	"github.com/arimatakao/comicfile/metadata"
)

type dirContainer struct {
	tempDir   string
	pageIndex int
}

type dirReader struct {
	pages    []image.Image
	errPages int
}

// newDirContainer creates a directory-backed container with a temporary
// staging directory for page files.
func newDirContainer() (*dirContainer, error) {
	tempDir, err := os.MkdirTemp("", "mdxdirfiles")
	if err != nil {
		return nil, err
	}

	return &dirContainer{
		tempDir:   tempDir,
		pageIndex: 1,
	}, nil
}

// openDirContainer creates a reader for the image files in path. Pages are
// ordered by filename, matching the naming convention used by dirContainer.
func openDirContainer(path string) (*dirReader, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}

	reader := &dirReader{pages: make([]image.Image, 0, len(entries))}
	for _, entry := range entries {
		if !entry.Type().IsRegular() {
			continue
		}

		page, err := readImage(filepath.Join(path, entry.Name()))
		if err != nil {
			reader.errPages++
			continue
		}
		reader.pages = append(reader.pages, page)
	}

	return reader, nil
}

// TotalPages returns the number of readable images in the directory.
func (d *dirReader) TotalPages() int {
	return len(d.pages)
}

// ErrPages returns the number of pages that could not be read.
func (d *dirReader) ErrPages() int {
	return d.errPages
}

// Page returns the decoded image at index.
func (d *dirReader) Page(index int) (image.Image, error) {
	if index < 0 || index >= len(d.pages) {
		return nil, ErrPageIndexOutOfRange
	}

	return d.pages[index], nil
}

func readImage(path string) (image.Image, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	page, _, err := image.Decode(file)
	return page, err
}

// WriteOnDiskAndClose copies staged page files into a unique directory in
// outputDir, then removes the temporary staging directory.
func (d *dirContainer) WriteOnDiskAndClose(outputDir, outputFileName string,
	m metadata.Metadata, chapterRange string) error {
	if err := os.MkdirAll(outputDir, os.ModePerm); err != nil {
		return err
	}

	outputPath := safeOutputDirPath(outputDir, outputFileName)
	if err := os.MkdirAll(outputPath, os.ModePerm); err != nil {
		return err
	}

	entries, err := os.ReadDir(d.tempDir)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		srcPath := filepath.Join(d.tempDir, entry.Name())
		dstPath := filepath.Join(outputPath, entry.Name())
		if err := copyFile(srcPath, dstPath); err != nil {
			return err
		}
	}

	return os.RemoveAll(d.tempDir)
}

// AddPage writes image bytes to the staging directory using the next
// zero-padded page filename.
func (d *dirContainer) AddPage(fileExt string, imageBytes []byte) error {
	fileName := fmt.Sprintf("%02d.%s", d.pageIndex, fileExt)
	filePath := filepath.Join(d.tempDir, fileName)
	if err := os.WriteFile(filePath, imageBytes, os.ModePerm); err != nil {
		return err
	}

	d.pageIndex++
	return nil
}

// copyFile copies the complete contents of srcPath into a newly created file
// at dstPath.
func copyFile(srcPath, dstPath string) error {
	srcFile, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dstPath)
	if err != nil {
		return err
	}

	if _, err := io.Copy(dstFile, srcFile); err != nil {
		_ = dstFile.Close()
		return err
	}

	if err := dstFile.Close(); err != nil {
		return err
	}

	return nil
}

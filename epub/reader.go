package epub

import (
	"archive/zip"
	"encoding/xml"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"os"
	"path"
	"strings"
	"sync"

	"github.com/arimatakao/comicfile/internal/container"
	"github.com/arimatakao/comicfile/metadata"
)

type epubReader struct {
	mu       sync.RWMutex
	archive  *zip.ReadCloser
	pages    []*zip.File
	errPages int
	metadata metadata.Metadata
	closed   bool
}

type epubContainer struct {
	Rootfiles []epubRootfile `xml:"rootfiles>rootfile"`
}
type epubRootfile struct {
	FullPath string `xml:"full-path,attr"`
}
type epubPackage struct {
	Metadata epubPackageMetadata `xml:"metadata"`
	Manifest []epubManifestItem  `xml:"manifest>item"`
	Spine    []epubSpineItem     `xml:"spine>itemref"`
}
type epubPackageMetadata struct {
	Title       string   `xml:"title"`
	Creators    []string `xml:"creator"`
	Language    string   `xml:"language"`
	Description string   `xml:"description"`
}
type epubManifestItem struct {
	ID   string `xml:"id,attr"`
	Href string `xml:"href,attr"`
}
type epubSpineItem struct {
	IDRef string `xml:"idref,attr"`
}

// Open creates a reader for the page images referenced by an EPUB's spine.
func Open(filePath string) (*epubReader, error) {
	archive, err := zip.OpenReader(filePath)
	if err != nil {
		return nil, err
	}
	keepArchive := false
	defer func() {
		if !keepArchive {
			_ = archive.Close()
		}
	}()
	files := make(map[string]*zip.File, len(archive.File))
	for _, file := range archive.File {
		files[file.Name] = file
	}
	containerFile, ok := files["META-INF/container.xml"]
	if !ok {
		return nil, fmt.Errorf("EPUB container file is missing")
	}
	var containerDocument epubContainer
	if err := decodeXMLFile(containerFile, &containerDocument); err != nil {
		return nil, fmt.Errorf("read EPUB container: %w", err)
	}
	if len(containerDocument.Rootfiles) == 0 || containerDocument.Rootfiles[0].FullPath == "" {
		return nil, fmt.Errorf("EPUB package document is missing")
	}
	packagePath := path.Clean(containerDocument.Rootfiles[0].FullPath)
	packageFile, ok := files[packagePath]
	if !ok {
		return nil, fmt.Errorf("EPUB package document %q is missing", packagePath)
	}
	var packageDocument epubPackage
	if err := decodeXMLFile(packageFile, &packageDocument); err != nil {
		return nil, fmt.Errorf("read EPUB package: %w", err)
	}

	reader := &epubReader{
		archive: archive,
		pages:   make([]*zip.File, 0, len(packageDocument.Spine)),
		metadata: metadata.Metadata{
			CI: metadata.ComicInfoMetadata{
				Title:       packageDocument.Metadata.Title,
				Writer:      strings.Join(packageDocument.Metadata.Creators, ", "),
				LanguageISO: packageDocument.Metadata.Language,
				Summary:     packageDocument.Metadata.Description,
			},
			P: metadata.PlainMetadata{Authors: strings.Join(packageDocument.Metadata.Creators, ", ")},
		},
	}
	manifest := make(map[string]string, len(packageDocument.Manifest))
	for _, item := range packageDocument.Manifest {
		manifest[item.ID] = item.Href
	}
	for _, spineItem := range packageDocument.Spine {
		sectionHref, ok := manifest[spineItem.IDRef]
		if !ok {
			reader.errPages++
			continue
		}
		sectionPath := path.Clean(path.Join(path.Dir(packagePath), sectionHref))
		sectionFile, ok := files[sectionPath]
		if !ok {
			reader.errPages++
			continue
		}
		reader.readSection(sectionFile, sectionPath, files)
	}
	keepArchive = true
	return reader, nil
}

// TotalPages returns the number of readable images referenced by the EPUB spine.
func (e *epubReader) TotalPages() int {
	return len(e.pages)
}

// ErrPages returns the number of page references that could not be read.
func (e *epubReader) ErrPages() int {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.errPages
}

// Metadata returns metadata available in the EPUB package document.
func (e *epubReader) Metadata() *metadata.Metadata {
	return &e.metadata
}

// Page decodes and returns one page from the still-open EPUB archive.
func (e *epubReader) Page(index int) (image.Image, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	if index < 0 || index >= len(e.pages) {
		return nil, container.ErrPageIndexOutOfRange
	}
	if e.closed {
		return nil, os.ErrClosed
	}
	return decodeImageFile(e.pages[index])
}

// Close releases the open EPUB archive.
func (e *epubReader) Close() error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.closed {
		return nil
	}
	e.closed = true
	return e.archive.Close()
}

func (e *epubReader) readSection(sectionFile *zip.File, sectionPath string, files map[string]*zip.File) {
	reader, err := sectionFile.Open()
	if err != nil {
		e.errPages++
		return
	}
	defer reader.Close()

	decoder := xml.NewDecoder(reader)
	for {
		token, err := decoder.Token()
		if err == io.EOF {
			return
		}
		if err != nil {
			e.errPages++
			return
		}

		start, ok := token.(xml.StartElement)
		if !ok || start.Name.Local != "img" {
			continue
		}

		imagePath := imageSource(start)
		if imagePath == "" {
			e.errPages++
			continue
		}

		imageFile, ok := files[path.Clean(path.Join(path.Dir(sectionPath), imagePath))]
		if !ok {
			e.errPages++
			continue
		}

		if _, err := imageConfig(imageFile); err != nil {
			e.errPages++
			continue
		}

		e.pages = append(e.pages, imageFile)
	}
}

func imageSource(element xml.StartElement) string {
	for _, attribute := range element.Attr {
		if attribute.Name.Local == "src" {
			return strings.Split(attribute.Value, "#")[0]
		}
	}
	return ""
}

func decodeXMLFile(file *zip.File, value any) error {
	reader, err := file.Open()
	if err != nil {
		return err
	}
	defer reader.Close()

	return xml.NewDecoder(reader).Decode(value)
}

func decodeImageFile(file *zip.File) (image.Image, error) {
	reader, err := file.Open()
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	page, _, err := image.Decode(reader)
	return page, err
}

func imageConfig(file *zip.File) (image.Config, error) {
	reader, err := file.Open()
	if err != nil {
		return image.Config{}, err
	}
	defer reader.Close()
	config, _, err := image.DecodeConfig(reader)
	return config, err
}

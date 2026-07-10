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
	"path/filepath"
	"strings"

	"github.com/arimatakao/comicfile/internal/container"
	"github.com/arimatakao/comicfile/metadata"
	"github.com/go-shiori/go-epub"
)

const imageSectionTemplate = `<img src="%s" alt="%s" />`

type epubArchive struct {
	b          *epub.Epub
	tempDir    string
	filesPaths []string
	pageIndex  int
}

type epubReader struct {
	pages    []image.Image
	errPages int
	metadata metadata.Metadata
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

// New creates an EPUB builder and a temporary directory for page
// images that will be embedded during finalization.
func New() (*epubArchive, error) {
	book, err := epub.NewEpub("")
	if err != nil {
		return &epubArchive{}, err
	}

	dir, err := os.MkdirTemp("", "mdxepubfiles")
	if err != nil {
		return &epubArchive{}, err
	}

	return &epubArchive{
		b:          book,
		tempDir:    dir,
		filesPaths: []string{},
		pageIndex:  1,
	}, nil
}

// WriteOnDiskAndClose adds the staged images and metadata to the EPUB, writes
// it to a unique file in outputDir, and removes the temporary image directory.
func (e *epubArchive) WriteOnDiskAndClose(outputDir string, outputFileName string,
	m metadata.Metadata, chapterRange string) error {

	for i, filePath := range e.filesPaths {
		indexPage := fmt.Sprintf("%02d", i+1)
		imageEpubPath, err := e.b.AddImage(filePath, indexPage)
		if err != nil {
			if err = os.RemoveAll(e.tempDir); err != nil {
				return err
			}
			return err
		}
		sectionStr := fmt.Sprintf(imageSectionTemplate, imageEpubPath, indexPage)
		_, err = e.b.AddSection(sectionStr, indexPage, "", "")
		if err != nil {
			if err = os.RemoveAll(e.tempDir); err != nil {
				return err
			}
			return err
		}
	}

	bookTitle := fmt.Sprintf("%s vol%s ch%s", m.CI.Title, m.CI.Volume, m.CI.Number)
	if chapterRange != "" {
		bookTitle = fmt.Sprintf("%s ch%s", m.CI.Title, chapterRange)
	}
	e.b.SetTitle(bookTitle)

	authors := m.P.Authors + " | " + m.P.Artists
	e.b.SetAuthor(authors)

	e.b.SetLang(m.CI.LanguageISO)

	e.b.SetDescription(m.CI.Summary)

	err := os.MkdirAll(outputDir, os.ModePerm)
	if err != nil {
		return err
	}

	outputPath := container.SafeOutputPath(outputDir, outputFileName, "epub")

	err = e.b.Write(outputPath)
	if err != nil {
		return err
	}

	return os.RemoveAll(e.tempDir)
}

// AddPage stages image bytes as the next zero-padded page file for inclusion
// in the EPUB during finalization.
func (e *epubArchive) AddPage(fileExt string, imageBytes []byte) error {
	fileName := fmt.Sprintf("%02d.%s", e.pageIndex, fileExt)
	filePath := filepath.Join(e.tempDir, fileName)
	err := os.WriteFile(filePath, imageBytes, os.ModePerm)
	if err != nil {
		return err
	}

	e.filesPaths = append(e.filesPaths, filePath)

	e.pageIndex++
	return nil
}

// Open creates a reader for the page images referenced by an EPUB's spine.
func Open(filePath string) (*epubReader, error) {
	archive, err := zip.OpenReader(filePath)
	if err != nil {
		return nil, err
	}
	defer archive.Close()

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
		pages: make([]image.Image, 0, len(packageDocument.Spine)),
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

	return reader, nil
}

// TotalPages returns the number of readable images referenced by the EPUB spine.
func (e *epubReader) TotalPages() int {
	return len(e.pages)
}

// ErrPages returns the number of page references that could not be read.
func (e *epubReader) ErrPages() int {
	return e.errPages
}

// Metadata returns metadata available in the EPUB package document.
func (e *epubReader) Metadata() *metadata.Metadata {
	return &e.metadata
}

// Page returns the decoded image at index.
func (e *epubReader) Page(index int) (image.Image, error) {
	if index < 0 || index >= len(e.pages) {
		return nil, container.ErrPageIndexOutOfRange
	}

	return e.pages[index], nil
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

		page, err := decodeImageFile(imageFile)
		if err != nil {
			e.errPages++
			continue
		}
		e.pages = append(e.pages, page)
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

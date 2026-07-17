package pdf

import (
	"errors"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/arimatakao/comicfile/internal/container"
	"github.com/arimatakao/comicfile/metadata"
	"github.com/pdfcpu/pdfcpu/pkg/api"
)

var errPageImageCount = errors.New("PDF page does not contain exactly one image")

type pdfReader struct {
	mu       sync.RWMutex
	path     string
	pages    int
	errPages int
	metadata metadata.Metadata
}

// Open creates a reader for PDF files containing one embedded image per page.
// Embedded images are extracted lazily by Page.
func Open(path string) (*pdfReader, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	info, err := api.PDFInfo(file, path, nil, false, nil)
	if err != nil {
		return nil, err
	}
	pages, err := api.PageCountFile(path)
	if err != nil {
		return nil, err
	}
	properties := make(map[string]string, len(info.Properties)+5)
	for key, value := range info.Properties {
		properties[key] = value
	}
	properties["Title"] = info.Title
	properties["Author"] = info.Author
	properties["Subject"] = info.Subject
	properties["Creator"] = info.Creator
	properties["Keywords"] = strings.Join(info.Keywords, ", ")
	return &pdfReader{path: path, pages: pages, metadata: metadataFromProperties(properties)}, nil
}

func (p *pdfReader) TotalPages() int { return p.pages }

func (p *pdfReader) ErrPages() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.errPages
}

func (p *pdfReader) Metadata() *metadata.Metadata { return &p.metadata }

// Page extracts and decodes one embedded image from the requested PDF page.
func (p *pdfReader) Page(index int) (image.Image, error) {
	if index < 0 || index >= p.pages {
		return nil, container.ErrPageIndexOutOfRange
	}
	file, err := os.Open(p.path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	images, err := api.ExtractImagesRaw(file, []string{strconv.Itoa(index + 1)}, nil)
	if err != nil {
		p.recordError()
		return nil, err
	}
	if len(images) != 1 || len(images[0]) != 1 {
		p.recordError()
		return nil, errPageImageCount
	}
	for _, raw := range images[0] {
		page, _, err := image.Decode(raw)
		if err != nil {
			p.recordError()
		}
		return page, err
	}
	return nil, errPageImageCount
}

// Close implements comicfile.ContainerReader. PDF pages are opened per call.
func (*pdfReader) Close() error { return nil }

func (p *pdfReader) recordError() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.errPages++
}

func metadataFromProperties(properties map[string]string) metadata.Metadata {
	return metadata.Metadata{
		CBI: metadata.ComicBookMetadata{
			AppID: properties["Creator"],
			ComicBookInfoData: metadata.ComicBookInfo{
				Title: properties["Subject"],
			},
		},
		CI: metadata.ComicInfoMetadata{
			Title:   properties["Title"],
			Summary: properties["Subject"],
		},
		P: metadata.PlainMetadata{
			Authors: properties["Author"],
			Tags:    properties["Keywords"],
		},
	}
}

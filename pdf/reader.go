package pdf

import (
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"sort"

	"github.com/arimatakao/comicfile/internal/container"
	"github.com/arimatakao/comicfile/metadata"
	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

type pdfReader struct {
	pages    []image.Image
	errPages int
	metadata metadata.Metadata
}

type extractedPageImage struct {
	pageNumber int
	image      model.Image
}

// Open creates a reader for PDF files containing one embedded image per page.
func Open(path string) (*pdfReader, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	defer file.Close()

	reader := &pdfReader{}
	if properties, err := api.Properties(file, nil); err == nil {
		reader.metadata = metadataFromProperties(properties)
	}
	if _, err := file.Seek(0, 0); err != nil {
		return nil, err
	}

	images, err := api.ExtractImagesRaw(file, nil, nil)
	if err != nil {
		return nil, err
	}

	extractedImages := make([]extractedPageImage, 0, len(images))
	for _, pageImages := range images {
		if len(pageImages) != 1 {
			reader.errPages++
			continue
		}

		for _, pageImage := range pageImages {
			extractedImages = append(extractedImages, extractedPageImage{pageNumber: pageImage.PageNr, image: pageImage})
		}
	}
	sort.Slice(extractedImages, func(i, j int) bool {
		return extractedImages[i].pageNumber < extractedImages[j].pageNumber
	})

	reader.pages = make([]image.Image, 0, len(extractedImages))
	for _, pageImage := range extractedImages {
		page, _, err := image.Decode(pageImage.image)
		if err != nil {
			reader.errPages++
			continue
		}
		reader.pages = append(reader.pages, page)
	}

	return reader, nil
}

// TotalPages returns the number of readable one-image PDF pages.
func (p *pdfReader) TotalPages() int {
	return len(p.pages)
}

// ErrPages returns the number of PDF pages that could not be read as one image.
func (p *pdfReader) ErrPages() int {
	return p.errPages
}

// Metadata returns metadata available in the PDF document information dictionary.
func (p *pdfReader) Metadata() *metadata.Metadata {
	return &p.metadata
}

// Page returns the decoded image at index.
func (p *pdfReader) Page(index int) (image.Image, error) {
	if index < 0 || index >= len(p.pages) {
		return nil, container.ErrPageIndexOutOfRange
	}

	return p.pages[index], nil
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
		},
	}
}

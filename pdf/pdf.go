package pdf

import (
	"bytes"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"time"

	"github.com/arimatakao/comicfile/internal/container"
	"github.com/arimatakao/comicfile/metadata"
	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/signintech/gopdf"
)

type pdfFile struct {
	pdf *gopdf.GoPdf
}

type pdfReader struct {
	pages    []image.Image
	errPages int
	metadata metadata.Metadata
}

// New creates a PDF container configured to preserve original image
// data and accept pages of arbitrary dimensions.
func New() (pdfFile, error) {

	pdf := new(gopdf.GoPdf)
	pdf.Start(gopdf.Config{
		PageSize: *gopdf.PageSizeA4,
	})

	pdf.SetNoCompression()

	return pdfFile{
		pdf: pdf,
	}, nil
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

	reader.pages = make([]image.Image, 0, len(images))
	for _, pageImages := range images {
		if len(pageImages) != 1 {
			reader.errPages++
			continue
		}

		for _, pageImage := range pageImages {
			page, _, err := image.Decode(pageImage)
			if err != nil {
				reader.errPages++
				continue
			}
			reader.pages = append(reader.pages, page)
		}
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

// WriteOnDiskAndClose applies document metadata, writes the PDF to a unique
// file in outputDir, and closes the PDF writer.
func (p pdfFile) WriteOnDiskAndClose(outputDir, outputFileName string,
	m metadata.Metadata, chapterRange string) error {
	author := m.P.Authors + " | " + m.P.Artists

	p.pdf.SetInfo(gopdf.PdfInfo{
		Title:        m.CI.Title,
		Author:       author,
		Subject:      m.CBI.ComicBookInfoData.Title,
		Creator:      m.CBI.AppID,
		Producer:     m.CBI.AppID,
		CreationDate: time.Now(),
	})

	err := os.MkdirAll(outputDir, os.ModePerm)
	if err != nil {
		return err
	}

	outputPath := container.SafeOutputPath(outputDir, outputFileName, "pdf")

	err = p.pdf.WritePdf(outputPath)
	if err != nil {
		return err
	}

	err = p.pdf.Close()
	if err != nil {
		return err
	}

	return nil
}

// AddPage decodes image dimensions, creates a matching PDF page, and places
// the image at its original size.
func (p pdfFile) AddPage(fileName string, imageBytes []byte) error {
	imgWidth, imgHeight, err := getImageDimensions(imageBytes)
	if err != nil {
		return err
	}

	imageReader := bytes.NewBuffer(imageBytes)

	p.pdf.AddPageWithOption(gopdf.PageOption{
		PageSize: &gopdf.Rect{
			W: imgWidth,
			H: imgHeight,
		},
	})

	imgH1, err := gopdf.ImageHolderByReader(imageReader)
	if err != nil {
		return err
	}
	if err := p.pdf.ImageByHolder(imgH1, 0, 0, &gopdf.Rect{
		W: float64(imgWidth),
		H: float64(imgHeight),
	}); err != nil {
		return err
	}

	return nil
}

// getImageDimensions returns the decoded pixel width and height of img.
func getImageDimensions(img []byte) (float64, float64, error) {
	buf := bytes.NewBuffer(img)
	config, _, err := image.DecodeConfig(buf)
	if err != nil {
		return 0, 0, err
	}
	return float64(config.Width), float64(config.Height), nil
}

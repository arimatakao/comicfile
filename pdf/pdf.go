package pdf

import (
	"bytes"
	"image"
	"os"

	"github.com/arimatakao/comicfile/internal/container"
	"github.com/arimatakao/comicfile/metadata"
	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/signintech/gopdf"
)

type pdfFile struct {
	pdf *gopdf.GoPdf
}

// New creates a PDF writer that preserves original image data and uses each
// image's dimensions for its page size.
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

// WriteOnDiskAndClose applies document metadata, writes the PDF to a unique
// file below outputDir, and closes the writer.
func (p pdfFile) WriteOnDiskAndClose(outputDir, outputFileName string,
	m metadata.Metadata, chapterRange string) error {

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

	return api.AddPropertiesFile(outputPath, outputPath, metadataProperties(m), nil)
}

func metadataProperties(m metadata.Metadata) map[string]string {
	properties := map[string]string{}
	for key, value := range map[string]string{
		"Title":    m.CI.Title,
		"Author":   m.P.Authors,
		"Subject":  m.CBI.ComicBookInfoData.Title,
		"Creator":  m.CBI.AppID,
		"Producer": m.CBI.AppID,
		"Keywords": m.P.Tags,
	} {
		if value != "" {
			properties[key] = value
		}
	}
	return properties
}

// AddPage creates a page matching the decoded image dimensions and places the
// image at its original size. fileName is ignored.
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

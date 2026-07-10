// Command generate-testdata creates container fixtures for package tests.
//
// Run it from the repository root:
//
//	go run ./testdata
package main

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/gif"
	"image/jpeg"
	"image/png"
	"log"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/arimatakao/comicfile"
	"github.com/arimatakao/comicfile/metadata"
)

type fixture struct {
	ext     string
	encode  func(*bytes.Buffer, image.Image) error
	image   image.Image
	content []byte
}

type containerCase struct {
	name     string
	format   string
	fixtures []fixture
	metadata metadata.Metadata
}

func main() {
	outputDir := filepath.Join(sourceDir(), "data")
	pngFixture := fixture{
		ext: "png",
		encode: func(buf *bytes.Buffer, img image.Image) error {
			return png.Encode(buf, img)
		},
		image: solidImage(3, 2, color.RGBA{R: 255, A: 255}),
	}
	jpegFixture := fixture{
		ext: "jpg",
		encode: func(buf *bytes.Buffer, img image.Image) error {
			return jpeg.Encode(buf, img, nil)
		},
		image: solidImage(2, 3, color.RGBA{G: 255, A: 255}),
	}
	gifFixture := fixture{
		ext: "gif",
		encode: func(buf *bytes.Buffer, img image.Image) error {
			return gif.Encode(buf, img, nil)
		},
		image: solidImage(1, 1, color.RGBA{B: 255, A: 255}),
	}
	invalidTextFixture := fixture{
		ext:     "txt",
		content: []byte("This is intentionally not an image.\n"),
	}
	cbzMetadata := metadata.Metadata{
		CBI: metadata.ComicBookMetadata{AppID: "comicfile-testdata"},
		CI:  metadata.ComicInfoMetadata{Title: "CBZ test chapter"},
	}
	epubMetadata := metadata.Metadata{
		CI: metadata.ComicInfoMetadata{
			Title: "EPUB test chapter", LanguageISO: "uk", Summary: "EPUB test summary",
		},
		P: metadata.PlainMetadata{Authors: "EPUB test author"},
	}
	pdfMetadata := metadata.Metadata{
		CBI: metadata.ComicBookMetadata{
			AppID:             "comicfile-testdata",
			ComicBookInfoData: metadata.ComicBookInfo{Title: "PDF test chapter images"},
		},
		CI: metadata.ComicInfoMetadata{Title: "PDF test chapter"},
		P:  metadata.PlainMetadata{Authors: "PDF test author"},
	}

	cases := []containerCase{
		{name: "dir-container-empty", format: comicfile.DIR_EXT},
		{
			name:     "dir-container-one-valid-png-3x2",
			format:   comicfile.DIR_EXT,
			fixtures: []fixture{pngFixture},
		},
		{
			name:     "dir-container-one-valid-jpeg-2x3",
			format:   comicfile.DIR_EXT,
			fixtures: []fixture{jpegFixture},
		},
		{
			name:     "dir-container-one-valid-gif-1x1",
			format:   comicfile.DIR_EXT,
			fixtures: []fixture{gifFixture},
		},
		{
			name:     "dir-container-valid-png-jpeg-gif",
			format:   comicfile.DIR_EXT,
			fixtures: []fixture{pngFixture, jpegFixture, gifFixture},
		},
		{
			name:     "dir-container-only-invalid-text",
			format:   comicfile.DIR_EXT,
			fixtures: []fixture{invalidTextFixture},
		},
		{
			name:     "dir-container-valid-png-and-invalid-text",
			format:   comicfile.DIR_EXT,
			fixtures: []fixture{pngFixture, invalidTextFixture},
		},
	}
	for _, testCase := range cases {
		testCase.name = strings.Replace(testCase.name, "dir-", "cbz-", 1)
		testCase.format = comicfile.CBZ_EXT
		testCase.metadata = cbzMetadata
		cases = append(cases, testCase)
	}
	for _, testCase := range cases {
		if !strings.HasPrefix(testCase.name, "dir-") {
			continue
		}
		testCase.name = strings.Replace(testCase.name, "dir-", "epub-", 1)
		testCase.format = comicfile.EPUB_EXT
		testCase.metadata = epubMetadata
		cases = append(cases, testCase)
	}
	cases = append(cases,
		containerCase{
			name:     "pdf-container-empty",
			format:   comicfile.PDF_EXT,
			metadata: pdfMetadata,
		},
		containerCase{
			name:     "pdf-container-two-valid-png-pages",
			format:   comicfile.PDF_EXT,
			fixtures: []fixture{pngFixture, pngFixture},
			metadata: pdfMetadata,
		},
	)

	for _, testCase := range cases {
		container, err := comicfile.NewContainer(testCase.format)
		if err != nil {
			log.Fatal(err)
		}

		for _, fixture := range testCase.fixtures {
			content, err := encodeFixture(fixture)
			if err != nil {
				log.Fatal(err)
			}
			if err := container.AddPage(fixture.ext, content); err != nil {
				log.Fatal(err)
			}
		}

		if err := container.WriteOnDiskAndClose(outputDir, testCase.name, testCase.metadata, ""); err != nil {
			log.Fatal(err)
		}
	}
}

func sourceDir() string {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		panic("could not find the testdata source directory")
	}
	return filepath.Dir(file)
}

func solidImage(width, height int, c color.Color) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, c)
		}
	}
	return img
}

func encodeFixture(fixture fixture) ([]byte, error) {
	if fixture.content != nil {
		return fixture.content, nil
	}

	var buf bytes.Buffer
	if err := fixture.encode(&buf, fixture.image); err != nil {
		return nil, fmt.Errorf("encode %s: %w", fixture.ext, err)
	}
	return buf.Bytes(), nil
}

// Command generate-testdata creates directory-container fixtures for package tests.
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
	fixtures []fixture
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

	cases := []containerCase{
		{name: "dir-container-empty"},
		{
			name:     "dir-container-one-valid-png-3x2",
			fixtures: []fixture{pngFixture},
		},
		{
			name:     "dir-container-one-valid-jpeg-2x3",
			fixtures: []fixture{jpegFixture},
		},
		{
			name:     "dir-container-one-valid-gif-1x1",
			fixtures: []fixture{gifFixture},
		},
		{
			name:     "dir-container-valid-png-jpeg-gif",
			fixtures: []fixture{pngFixture, jpegFixture, gifFixture},
		},
		{
			name:     "dir-container-only-invalid-text",
			fixtures: []fixture{invalidTextFixture},
		},
		{
			name:     "dir-container-valid-png-and-invalid-text",
			fixtures: []fixture{pngFixture, invalidTextFixture},
		},
	}

	for _, testCase := range cases {
		container, err := comicfile.NewContainer(comicfile.DIR_EXT)
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

		if err := container.WriteOnDiskAndClose(outputDir, testCase.name, metadata.Metadata{}, ""); err != nil {
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

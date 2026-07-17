// Command dir creates or reads the file directory.
package main

import (
	"bytes"
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"log"

	"github.com/arimatakao/comicfile"
	"github.com/arimatakao/comicfile/metadata"
)

const filePath = "file"

func main() {
	createMode := flag.Bool("c", false, "create the file directory")
	readMode := flag.Bool("r", false, "read the file directory")
	flag.Parse()
	if *createMode == *readMode || flag.NArg() != 0 {
		log.Fatal("use exactly one of -c or -r")
	}
	if *createMode {
		if err := create(); err != nil {
			log.Fatal(err)
		}
		return
	}
	if err := read(); err != nil {
		log.Fatal(err)
	}
}

func create() error {
	chapter, err := comicfile.NewContainer(comicfile.DIR_EXT)
	if err != nil {
		return err
	}
	images, err := CreateImages()
	if err != nil {
		return err
	}
	for _, imageBytes := range images {
		if err := chapter.AddPage("png", imageBytes); err != nil {
			return err
		}
	}
	if err := chapter.WriteOnDiskAndClose(".", filePath, exampleMetadata(), ""); err != nil {
		return err
	}
	fmt.Println("wrote", filePath)
	return nil
}

func read() error {
	chapter, err := comicfile.OpenContainer(filePath)
	if err != nil {
		return err
	}
	defer chapter.Close()
	printMetadata(chapter.Metadata())
	fmt.Printf("pages: %d\nunreadable pages: %d\n", chapter.TotalPages(), chapter.ErrPages())
	for i := 0; i < chapter.TotalPages(); i++ {
		page, err := chapter.Page(i)
		if err != nil {
			return err
		}
		bounds := page.Bounds()
		fmt.Printf("page %d: %dx%d\n", i+1, bounds.Dx(), bounds.Dy())
	}
	return nil
}

func exampleMetadata() metadata.Metadata {
	return metadata.Metadata{
		CI: metadata.ComicInfoMetadata{
			Title:       "Example chapter",
			Writer:      "Example author",
			Penciller:   "Example artist",
			LanguageISO: "en",
			PageCount:   3,
		},
		P: metadata.PlainMetadata{Authors: "Example author", Artists: "Example artist", Tags: "example"},
	}
}

func printMetadata(m *metadata.Metadata) {
	if m == nil {
		fmt.Println("metadata: unavailable")
		return
	}
	fmt.Printf("metadata:\n  title: %s\n  authors: %s\n  artists: %s\n  language: %s\n  tags: %s\n", m.CI.Title, m.P.Authors, m.P.Artists, m.CI.LanguageISO, m.P.Tags)
}

func CreateImages() ([][]byte, error) {
	images := make([][]byte, 0, 3)
	for _, pageSpec := range []struct {
		background color.Color
		width      int
		height     int
	}{
		{color.RGBA{R: 255, A: 255}, 100, 100},
		{color.RGBA{G: 255, A: 255}, 200, 200},
		{color.RGBA{B: 255, A: 255}, 300, 300},
	} {
		page := image.NewRGBA(image.Rect(0, 0, pageSpec.width, pageSpec.height))
		draw.Draw(page, page.Bounds(), image.NewUniform(pageSpec.background), image.Point{}, draw.Src)
		var encoded bytes.Buffer
		if err := png.Encode(&encoded, page); err != nil {
			return nil, err
		}
		images = append(images, encoded.Bytes())
	}
	return images, nil
}

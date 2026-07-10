// Command pdf creates or reads file.pdf.
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

const filePath = "file.pdf"

func main() {
	create := flag.Bool("c", false, "create file.pdf")
	read := flag.Bool("r", false, "read file.pdf")
	flag.Parse()
	if *create == *read || flag.NArg() != 0 {
		log.Fatal("use exactly one of -c or -r")
	}
	if *create {
		chapter, err := comicfile.NewContainer(comicfile.PDF_EXT)
		if err != nil {
			log.Fatal(err)
		}
		if err := addBluePages(chapter); err != nil {
			log.Fatal(err)
		}
		if err := chapter.WriteOnDiskAndClose(".", "file", metadata.Metadata{}, ""); err != nil {
			log.Fatal(err)
		}
		fmt.Println("wrote", filePath)
		return
	}

	chapter, err := comicfile.OpenContainer(filePath)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("pages: %d\nunreadable pages: %d\n", chapter.TotalPages(), chapter.ErrPages())
}

func addBluePages(chapter comicfile.ContainerWriter) error {
	for _, background := range []color.Color{
		color.RGBA{B: 255, A: 255},
		color.RGBA{G: 96, B: 200, A: 255},
	} {
		page := image.NewRGBA(image.Rect(0, 0, 100, 100))
		draw.Draw(page, page.Bounds(), image.NewUniform(background), image.Point{}, draw.Src)
		var encoded bytes.Buffer
		if err := png.Encode(&encoded, page); err != nil {
			return err
		}
		if err := chapter.AddPage("png", encoded.Bytes()); err != nil {
			return err
		}
	}
	return nil
}

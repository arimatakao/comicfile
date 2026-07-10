package pdf_test

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/arimatakao/comicfile"
	"github.com/arimatakao/comicfile/metadata"
	"github.com/arimatakao/comicfile/pdf"
)

func TestPDFWriteAndOpen(t *testing.T) {
	outputDir := t.TempDir()
	writer, err := pdf.New()
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	pages := []image.Point{{X: 3, Y: 2}, {X: 2, Y: 3}, {X: 4, Y: 5}}
	for _, page := range pages {
		if err := writer.AddPage("png", pngPage(t, page.X, page.Y)); err != nil {
			t.Fatalf("AddPage() error = %v", err)
		}
	}

	wantMetadata := metadata.Metadata{
		CBI: metadata.ComicBookMetadata{
			AppID:             "test-app",
			ComicBookInfoData: metadata.ComicBookInfo{Title: "Image title"},
		},
		CI: metadata.ComicInfoMetadata{Title: "Chapter"},
		P: metadata.PlainMetadata{
			Authors: "Author",
			Tags:    "tag",
		},
	}
	if err := writer.WriteOnDiskAndClose(outputDir, "chapter", wantMetadata, ""); err != nil {
		t.Fatalf("WriteOnDiskAndClose() error = %v", err)
	}

	reader, err := comicfile.OpenContainer(filepath.Join(outputDir, "chapter.pdf"))
	if err != nil {
		t.Fatalf("OpenContainer() error = %v", err)
	}
	if got, want := reader.TotalPages(), len(pages); got != want {
		t.Errorf("TotalPages() = %d, want %d", got, want)
	}
	if got := reader.ErrPages(); got != 0 {
		t.Errorf("ErrPages() = %d, want 0", got)
	}
	wantReadMetadata := metadata.Metadata{
		CBI: metadata.ComicBookMetadata{
			AppID:             "test-app",
			ComicBookInfoData: metadata.ComicBookInfo{Title: "Image title"},
		},
		CI: metadata.ComicInfoMetadata{
			Title:   "Chapter",
			Summary: "Image title",
		},
		P: metadata.PlainMetadata{
			Authors: "Author",
			Tags:    "tag",
		},
	}
	if got := reader.Metadata(); !reflect.DeepEqual(*got, wantReadMetadata) {
		t.Errorf("Metadata() = %#v, want %#v", *got, wantReadMetadata)
	}

	for index, want := range pages {
		page, err := reader.Page(index)
		if err != nil {
			t.Fatalf("Page(%d) error = %v", index, err)
		}
		if got := page.Bounds(); got.Dx() != want.X || got.Dy() != want.Y {
			t.Errorf("Page(%d) dimensions = %dx%d, want %dx%d", index, got.Dx(), got.Dy(), want.X, want.Y)
		}
	}

	for _, index := range []int{-1, len(pages)} {
		if _, err := reader.Page(index); err == nil {
			t.Errorf("Page(%d) error = nil, want out-of-range error", index)
		}
	}
}

func pngPage(t *testing.T, width, height int) []byte {
	t.Helper()

	page := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			page.Set(x, y, color.White)
		}
	}

	var encoded bytes.Buffer
	if err := png.Encode(&encoded, page); err != nil {
		t.Fatal(err)
	}
	return encoded.Bytes()
}

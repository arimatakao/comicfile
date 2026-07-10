package epub_test

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"path/filepath"
	"testing"

	comicfile "github.com/arimatakao/comicfile"
	"github.com/arimatakao/comicfile/epub"
	"github.com/arimatakao/comicfile/metadata"
)

func TestEPUBArchiveWriteAndOpen(t *testing.T) {
	outputDir := t.TempDir()
	archive, err := epub.New()
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	for _, size := range []image.Point{{X: 3, Y: 2}, {X: 2, Y: 3}} {
		if err := archive.AddPage("png", encodePNG(t, size.X, size.Y)); err != nil {
			t.Fatalf("AddPage() error = %v", err)
		}
	}
	metadataToWrite := metadata.Metadata{CI: metadata.ComicInfoMetadata{
		Title: "Chapter", LanguageISO: "uk", Summary: "Summary",
	}, P: metadata.PlainMetadata{Authors: "Author"}}
	if err := archive.WriteOnDiskAndClose(outputDir, "chapter", metadataToWrite, ""); err != nil {
		t.Fatalf("WriteOnDiskAndClose() error = %v", err)
	}

	reader, err := comicfile.OpenContainer(filepath.Join(outputDir, "chapter.epub"))
	if err != nil {
		t.Fatalf("OpenContainer() error = %v", err)
	}
	if got := reader.TotalPages(); got != 2 {
		t.Errorf("TotalPages() = %d, want 2", got)
	}
	if got := reader.ErrPages(); got != 0 {
		t.Errorf("ErrPages() = %d, want 0", got)
	}
	wantMetadata := metadata.Metadata{CI: metadata.ComicInfoMetadata{
		Title: "Chapter vol ch", Writer: "Author | ", LanguageISO: "uk", Summary: "Summary",
	}, P: metadata.PlainMetadata{Authors: "Author | "}}
	if got := reader.Metadata(); got == nil || got.CI.Title != wantMetadata.CI.Title || got.CI.Writer != wantMetadata.CI.Writer || got.CI.LanguageISO != wantMetadata.CI.LanguageISO || got.CI.Summary != wantMetadata.CI.Summary || got.P.Authors != wantMetadata.P.Authors {
		t.Errorf("Metadata() = %#v, want EPUB metadata", got)
	}
	for index, size := range []image.Point{{X: 3, Y: 2}, {X: 2, Y: 3}} {
		page, err := reader.Page(index)
		if err != nil {
			t.Fatalf("Page(%d) error = %v", index, err)
		}
		if got := page.Bounds(); got.Dx() != size.X || got.Dy() != size.Y {
			t.Errorf("Page(%d) dimensions = %dx%d, want %dx%d", index, got.Dx(), got.Dy(), size.X, size.Y)
		}
	}
	for _, index := range []int{-1, 2} {
		if _, err := reader.Page(index); err == nil {
			t.Errorf("Page(%d) error = nil, want out-of-range error", index)
		}
	}
}

func encodePNG(t *testing.T, width, height int) []byte {
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

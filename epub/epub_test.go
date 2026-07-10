package epub_test

import (
	"bytes"
	"image"
	"image/color"
	"image/gif"
	"image/jpeg"
	"image/png"
	"path/filepath"
	"testing"

	comicfile "github.com/arimatakao/comicfile"
	"github.com/arimatakao/comicfile/epub"
	"github.com/arimatakao/comicfile/metadata"
)

type pageSpec struct {
	ext    string
	format string
	width  int
	height int
}

func TestEPUBArchiveWriteAndOpen(t *testing.T) {
	outputDir := t.TempDir()
	archive, err := epub.New()
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	pages := []pageSpec{
		{ext: "png", format: "png", width: 3, height: 2},
		{ext: "jpg", format: "jpeg", width: 2, height: 3},
		{ext: "gif", format: "gif", width: 1, height: 1},
	}
	for _, page := range pages {
		if err := archive.AddPage(page.ext, encodePage(t, page)); err != nil {
			t.Fatalf("AddPage(%q) error = %v", page.ext, err)
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
	if got, want := reader.TotalPages(), len(pages); got != want {
		t.Errorf("TotalPages() = %d, want %d", got, want)
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
	for index, size := range pages {
		page, err := reader.Page(index)
		if err != nil {
			t.Fatalf("Page(%d) error = %v", index, err)
		}
		if got := page.Bounds(); got.Dx() != size.width || got.Dy() != size.height {
			t.Errorf("Page(%d) dimensions = %dx%d, want %dx%d", index, got.Dx(), got.Dy(), size.width, size.height)
		}
	}
	for _, index := range []int{-1, len(pages)} {
		if _, err := reader.Page(index); err == nil {
			t.Errorf("Page(%d) error = nil, want out-of-range error", index)
		}
	}
}

func TestOpenEPUBContainerSkipsUnreadablePages(t *testing.T) {
	outputDir := t.TempDir()
	archive, err := epub.New()
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if err := archive.AddPage("png", encodePage(t, pageSpec{format: "png", width: 1, height: 1})); err != nil {
		t.Fatalf("AddPage() error = %v", err)
	}
	if err := archive.AddPage("txt", []byte("not an image")); err != nil {
		t.Fatalf("AddPage() error = %v", err)
	}
	if err := archive.WriteOnDiskAndClose(outputDir, "chapter", metadata.Metadata{}, ""); err != nil {
		t.Fatalf("WriteOnDiskAndClose() error = %v", err)
	}

	reader, err := comicfile.OpenContainer(filepath.Join(outputDir, "chapter.epub"))
	if err != nil {
		t.Fatalf("OpenContainer() error = %v", err)
	}
	if got := reader.TotalPages(); got != 1 {
		t.Errorf("TotalPages() = %d, want 1", got)
	}
	if got := reader.ErrPages(); got != 1 {
		t.Errorf("ErrPages() = %d, want 1", got)
	}
}

func encodePage(t *testing.T, spec pageSpec) []byte {
	t.Helper()
	page := image.NewRGBA(image.Rect(0, 0, spec.width, spec.height))
	for y := 0; y < spec.height; y++ {
		for x := 0; x < spec.width; x++ {
			page.Set(x, y, color.White)
		}
	}

	var encoded bytes.Buffer
	var err error
	switch spec.format {
	case "png":
		err = png.Encode(&encoded, page)
	case "jpeg":
		err = jpeg.Encode(&encoded, page, nil)
	case "gif":
		err = gif.Encode(&encoded, page, nil)
	default:
		t.Fatalf("unsupported test image format %q", spec.format)
	}
	if err != nil {
		t.Fatal(err)
	}
	return encoded.Bytes()
}

package cbz_test

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
	"github.com/arimatakao/comicfile/cbz"
	"github.com/arimatakao/comicfile/metadata"
)

type pageSpec struct {
	ext    string
	format string
	width  int
	height int
}

func TestCBZArchiveWriteAndOpen(t *testing.T) {
	outputDir := t.TempDir()
	container, err := cbz.New()
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	specs := []pageSpec{
		{ext: "png", format: "png", width: 3, height: 2},
		{ext: "jpg", format: "jpeg", width: 2, height: 3},
		{ext: "gif", format: "gif", width: 1, height: 1},
	}
	addPages(t, container, specs)
	wantMetadata := metadata.Metadata{
		CBI: metadata.ComicBookMetadata{AppID: "test-app"},
		CI:  metadata.ComicInfoMetadata{Title: "Chapter"},
	}
	if err := container.WriteOnDiskAndClose(outputDir, "chapter", wantMetadata, ""); err != nil {
		t.Fatalf("WriteOnDiskAndClose() error = %v", err)
	}

	reader, err := comicfile.OpenContainer(filepath.Join(outputDir, "chapter.cbz"))
	if err != nil {
		t.Fatalf("OpenContainer() error = %v", err)
	}
	if got, want := reader.TotalPages(), len(specs); got != want {
		t.Errorf("TotalPages() = %d, want %d", got, want)
	}
	if got := reader.ErrPages(); got != 0 {
		t.Errorf("ErrPages() = %d, want 0", got)
	}
	if got := reader.Metadata(); got == nil || got.CBI.AppID != wantMetadata.CBI.AppID || got.CI.Title != wantMetadata.CI.Title {
		t.Errorf("Metadata() = %#v, want metadata with app ID %q and title %q", got, wantMetadata.CBI.AppID, wantMetadata.CI.Title)
	}
	for index, page := range specs {
		assertPageDimensions(t, reader, index, page.width, page.height)
	}
	assertOutOfRangePages(t, reader, len(specs))
}

func addPages(t *testing.T, container comicfile.ContainerWriter, specs []pageSpec) {
	t.Helper()
	for _, spec := range specs {
		if err := container.AddPage(spec.ext, encodePage(t, spec)); err != nil {
			t.Fatalf("AddPage(%q) error = %v", spec.ext, err)
		}
	}
}

func encodePage(t *testing.T, spec pageSpec) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, spec.width, spec.height))
	for y := 0; y < spec.height; y++ {
		for x := 0; x < spec.width; x++ {
			img.Set(x, y, color.White)
		}
	}
	var buf bytes.Buffer
	var err error
	switch spec.format {
	case "png":
		err = png.Encode(&buf, img)
	case "jpeg":
		err = jpeg.Encode(&buf, img, nil)
	case "gif":
		err = gif.Encode(&buf, img, nil)
	}
	if err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func assertPageDimensions(t *testing.T, reader comicfile.ContainerReader, index, width, height int) {
	t.Helper()
	page, err := reader.Page(index)
	if err != nil {
		t.Fatal(err)
	}
	if got := page.Bounds(); got.Dx() != width || got.Dy() != height {
		t.Errorf("Page(%d) dimensions = %dx%d, want %dx%d", index, got.Dx(), got.Dy(), width, height)
	}
}

func assertOutOfRangePages(t *testing.T, reader comicfile.ContainerReader, pages int) {
	t.Helper()
	for _, index := range []int{-1, pages} {
		if _, err := reader.Page(index); err == nil {
			t.Errorf("Page(%d) error = nil, want out-of-range error", index)
		}
	}
}

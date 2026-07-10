package comicfile

import (
	"path/filepath"
	"testing"

	"github.com/arimatakao/comicfile/metadata"
)

func TestCBZArchiveWriteAndOpen(t *testing.T) {
	outputDir := t.TempDir()
	container, err := newCBZArchive()
	if err != nil {
		t.Fatalf("newCBZArchive() error = %v", err)
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

	reader, err := OpenContainer(filepath.Join(outputDir, "chapter.cbz"))
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

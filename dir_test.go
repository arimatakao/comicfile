package comicfile

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/gif"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/arimatakao/comicfile/metadata"
)

type pageSpec struct {
	ext    string
	format string
	width  int
	height int
}

func TestDirContainerWriteAndOpen(t *testing.T) {
	tests := []struct {
		name           string
		outputName     string
		existingOutput bool
		wantOutputName string
		pages          []pageSpec
	}{
		{
			name:           "writes an empty container",
			outputName:     "empty",
			wantOutputName: "empty",
		},
		{
			name:           "writes pages and removes the staging directory",
			outputName:     "chapter",
			wantOutputName: "chapter",
			pages: []pageSpec{
				{ext: "png", format: "png", width: 3, height: 2},
				{ext: "jpg", format: "jpeg", width: 2, height: 3},
				{ext: "gif", format: "gif", width: 1, height: 1},
			},
		},
		{
			name:           "sanitizes the output directory name",
			outputName:     "chapter/with:invalid?characters",
			wantOutputName: "chapter_with_invalid_characters",
			pages: []pageSpec{
				{ext: "png", format: "png", width: 1, height: 1},
			},
		},
		{
			name:           "uses a suffix when output directory exists",
			outputName:     "chapter",
			existingOutput: true,
			wantOutputName: "chapter (1)",
			pages: []pageSpec{
				{ext: "png", format: "png", width: 1, height: 1},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			outputDir := t.TempDir()
			if tt.existingOutput {
				if err := os.Mkdir(filepath.Join(outputDir, tt.outputName), 0o755); err != nil {
					t.Fatal(err)
				}
			}

			container, err := newDirContainer()
			if err != nil {
				t.Fatalf("newDirContainer() error = %v", err)
			}

			pages := addPages(t, container, tt.pages)
			if err := container.WriteOnDiskAndClose(outputDir, tt.outputName, metadata.Metadata{}, ""); err != nil {
				t.Fatalf("WriteOnDiskAndClose() error = %v", err)
			}
			assertNotExists(t, container.tempDir)

			chapterDir := filepath.Join(outputDir, tt.wantOutputName)
			for index, page := range pages {
				name := fmt.Sprintf("%02d.%s", index+1, tt.pages[index].ext)
				assertFileContents(t, filepath.Join(chapterDir, name), page)
			}

			reader, err := OpenContainer(chapterDir)
			if err != nil {
				t.Fatalf("OpenContainer() error = %v", err)
			}
			if got, want := reader.TotalPages(), len(tt.pages); got != want {
				t.Errorf("TotalPages() = %d, want %d", got, want)
			}
			if got := reader.ErrPages(); got != 0 {
				t.Errorf("ErrPages() = %d, want 0", got)
			}

			for index, page := range tt.pages {
				assertPageDimensions(t, reader, index, page.width, page.height)
			}
			assertOutOfRangePages(t, reader, len(tt.pages))
		})
	}
}

func TestOpenDirContainer(t *testing.T) {
	tests := []struct {
		name         string
		pages        []pageSpec
		invalidFiles int
		addSubdir    bool
		wantPages    int
		wantErrPages int
	}{
		{
			name: "reads valid image files",
			pages: []pageSpec{
				{ext: "png", format: "png", width: 1, height: 1},
				{ext: "jpg", format: "jpeg", width: 2, height: 1},
				{ext: "gif", format: "gif", width: 1, height: 2},
			},
			wantPages: 3,
		},
		{
			name:         "opens an empty directory",
			wantPages:    0,
			wantErrPages: 0,
		},
		{
			name: "skips invalid files and directories",
			pages: []pageSpec{
				{ext: "png", format: "png", width: 1, height: 1},
			},
			invalidFiles: 1,
			addSubdir:    true,
			wantPages:    1,
			wantErrPages: 1,
		},
		{
			name:         "reports every invalid regular file",
			invalidFiles: 2,
			wantPages:    0,
			wantErrPages: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			for index, page := range tt.pages {
				path := filepath.Join(dir, fmt.Sprintf("%02d.%s", index+1, page.ext))
				if err := os.WriteFile(path, encodePage(t, page), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			for index := range tt.invalidFiles {
				path := filepath.Join(dir, fmt.Sprintf("invalid-%02d.txt", index+1))
				if err := os.WriteFile(path, []byte("not an image"), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			if tt.addSubdir {
				if err := os.Mkdir(filepath.Join(dir, "subdirectory"), 0o755); err != nil {
					t.Fatal(err)
				}
			}

			reader, err := OpenContainer(dir)
			if err != nil {
				t.Fatalf("OpenContainer() error = %v", err)
			}
			if got := reader.TotalPages(); got != tt.wantPages {
				t.Errorf("TotalPages() = %d, want %d", got, tt.wantPages)
			}
			if got := reader.ErrPages(); got != tt.wantErrPages {
				t.Errorf("ErrPages() = %d, want %d", got, tt.wantErrPages)
			}
			assertOutOfRangePages(t, reader, tt.wantPages)
		})
	}
}

func TestOpenContainerRejectsRegularFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "not-a-directory")
	if err := os.WriteFile(path, []byte("not a container"), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := OpenContainer(path); !errors.Is(err, ErrExtensionNotSupport) {
		t.Errorf("OpenContainer() error = %v, want ErrExtensionNotSupport", err)
	}
}

func addPages(t *testing.T, container *dirContainer, specs []pageSpec) [][]byte {
	t.Helper()
	pages := make([][]byte, 0, len(specs))
	for _, spec := range specs {
		page := encodePage(t, spec)
		if err := container.AddPage(spec.ext, page); err != nil {
			t.Fatalf("AddPage(%q) error = %v", spec.ext, err)
		}
		pages = append(pages, page)
	}
	return pages
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
	default:
		t.Fatalf("unsupported test image format %q", spec.format)
	}
	if err != nil {
		t.Fatalf("encode %s: %v", spec.format, err)
	}
	return buf.Bytes()
}

func assertNotExists(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("%q still exists or could not be checked: %v", path, err)
	}
}

func assertFileContents(t *testing.T, path string, want []byte) {
	t.Helper()
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%q) error = %v", path, err)
	}
	if !bytes.Equal(got, want) {
		t.Errorf("contents of %q differ from added page", path)
	}
}

func assertPageDimensions(t *testing.T, reader ContainerReader, index, width, height int) {
	t.Helper()
	page, err := reader.Page(index)
	if err != nil {
		t.Fatalf("Page(%d) error = %v", index, err)
	}
	if got := page.Bounds(); got.Dx() != width || got.Dy() != height {
		t.Errorf("Page(%d) dimensions = %dx%d, want %dx%d", index, got.Dx(), got.Dy(), width, height)
	}
}

func assertOutOfRangePages(t *testing.T, reader ContainerReader, pages int) {
	t.Helper()
	for _, index := range []int{-1, pages} {
		if _, err := reader.Page(index); !errors.Is(err, ErrPageIndexOutOfRange) {
			t.Errorf("Page(%d) error = %v, want ErrPageIndexOutOfRange", index, err)
		}
	}
}

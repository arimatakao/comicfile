// Package comicfile writes and reads comic or manga chapters stored as CBZ,
// PDF, EPUB, or directories of image files.
//
// A typical flow is:
//  1. Create a writer with NewContainer.
//  2. Add image pages with ContainerWriter.AddPage.
//  3. Write the completed container with ContainerWriter.WriteOnDiskAndClose.
package comicfile

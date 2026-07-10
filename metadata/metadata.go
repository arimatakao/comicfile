package metadata

import (
	"encoding/xml"
	"fmt"
	"time"
)

// Metadata contains the metadata representations written by the supported
// comic container formats.
type Metadata struct {
	// CBI is metadata in the ComicBookInfo format.
	CBI ComicBookMetadata
	// CI is metadata in the ComicInfo.xml format.
	CI ComicInfoMetadata
	// P contains metadata shared by formats that do not use a comic-specific
	// schema.
	P PlainMetadata
}

// PlainMetadata contains author, artist, and tag values for formats with no
// dedicated comic metadata schema.
type PlainMetadata struct {
	Authors string
	Artists string
	Tags    string
}

// ComicInfoMetadata is the ComicInfo.xml metadata schema used by ComicRack and
// CBZ readers.
type ComicInfoMetadata struct {
	XMLName     xml.Name `xml:"ComicInfo"`
	Title       string   `xml:"Title"`
	Number      string   `xml:"Number,omitempty"`
	Volume      string   `xml:"Volume,omitempty"`
	Year        int      `xml:"Year"`
	Writer      string   `xml:"Writer"`
	Penciller   string   `xml:"Penciller"`
	Inker       string   `xml:"Inker"`
	Publisher   string   `xml:"Publisher"`
	PageCount   int      `xml:"PageCount"`
	LanguageISO string   `xml:"LanguageISO"`
	Format      string   `xml:"Format"`
	Manga       string   `xml:"Manga"`
	Summary     string   `xml:"Summary"`
}

// ComicBookMetadata is the ComicBookInfo metadata stored in a CBZ archive
// comment.
type ComicBookMetadata struct {
	AppID             string        `json:"appID"`
	LastModified      string        `json:"lastModified"`
	ComicBookInfoData ComicBookInfo `json:"ComicBookInfo/1.0"`
}

// ComicBookInfo contains the ComicBookInfo 1.0 fields embedded in
// ComicBookMetadata.
type ComicBookInfo struct {
	Series    string   `json:"series"`
	Title     string   `json:"title"`
	Publisher string   `json:"publisher"`
	Issue     string   `json:"issue"`
	Volume    string   `json:"volume"`
	Language  string   `json:"language"`
	Credits   []Credit `json:"credits"`
	Tags      []string `json:"tags"`
}

// Credit identifies a person and their role in creating a comic.
type Credit struct {
	Person string `json:"person"`
	Role   string `json:"role"`
}

// MangaProvider supplies series-level information for NewMetadata.
type MangaProvider interface {
	Title(language string) string
	Description(language string) string
	Publisher() string
	Year() int
	AuthorsArr() []string
	Authors() string
	ArtistsArr() []string
	Artists() string
	TagsArr() []string
	Tags() string
	LinksArr() []string
	Links() string
}

// ChapterProvider supplies chapter-level information for NewMetadata.
type ChapterProvider interface {
	Title() string
	Number() string
	Volume() string
	Language() string
	PagesCount() int
}

// NewMetadata builds ComicBookInfo, ComicInfo.xml, and plain metadata from
// series and chapter providers for use by the supported output containers.
func NewMetadata(appId string, m MangaProvider, c ChapterProvider) Metadata {

	credits := []Credit{}
	for _, au := range m.AuthorsArr() {
		credit := Credit{
			Person: au,
			Role:   "Writer",
		}
		credits = append(credits, credit)
	}
	for _, ar := range m.ArtistsArr() {
		credit := Credit{
			Person: ar,
			Role:   "Artist",
		}
		credits = append(credits, credit)
	}

	mangaTitle := fmt.Sprintf("%s | %s vol%s ch%s",
		c.Language(), m.Title("en"), c.Volume(), c.Number())

	mangaDescription := m.Description("en") + "<br>Read or Buy here:<br>"
	for _, l := range m.LinksArr() {
		mangaDescription += l + "<br>"
	}

	metadata := Metadata{
		CBI: ComicBookMetadata{
			AppID:        appId,
			LastModified: time.Now().UTC().String(),
			ComicBookInfoData: ComicBookInfo{
				Series:    mangaTitle,
				Title:     c.Title(),
				Publisher: m.Publisher(),
				Issue:     c.Number(),
				Volume:    c.Volume(),
				Language:  c.Language(),
				Credits:   credits,
				Tags:      m.TagsArr(),
			},
		},
		CI: ComicInfoMetadata{
			XMLName:     xml.Name{Local: "ComicInfo"},
			Title:       mangaTitle,
			Number:      c.Number(),
			Volume:      c.Volume(),
			Year:        m.Year(),
			Writer:      m.Authors(),
			Penciller:   m.Artists(),
			Inker:       m.Artists(),
			Publisher:   m.Publisher(),
			PageCount:   c.PagesCount(),
			LanguageISO: c.Language(),
			Format:      "Comic Book",
			Manga:       "No",
			Summary:     mangaDescription,
		},
		P: PlainMetadata{
			Authors: m.Authors(),
			Artists: m.Artists(),
			Tags:    m.Tags(),
		},
	}

	return metadata
}

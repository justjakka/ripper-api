package ripper

import (
	"io"
	"regexp"
	"time"

	"github.com/abema/go-mp4"
)

const (
	defaultId   = "0"
	prefetchKey = "skd://itunes.apple.com/P000000000/s1/e1"
)

var (
	ForbiddenNames = regexp.MustCompile(`[/\\<>:"|?*]`)
)

type SampleInfo struct {
	data      []byte
	duration  uint32
	descIndex uint32
}

type SongInfo struct {
	r         io.ReadSeeker
	alacParam *Alac
	samples   []SampleInfo
}

type ApiResult struct {
	Data []SongData `json:"data"`
}

type SongAttributes struct {
	ArtistName        string   `json:"artistName"`
	DiscNumber        int      `json:"discNumber"`
	GenreNames        []string `json:"genreNames"`
	ExtendedAssetUrls struct {
		EnhancedHls string `json:"enhancedHls"`
	} `json:"extendedAssetUrls"`
	IsMasteredForItunes bool   `json:"isMasteredForItunes"`
	ReleaseDate         string `json:"releaseDate"`
	Name                string `json:"name"`
	Isrc                string `json:"isrc"`
	AlbumName           string `json:"albumName"`
	TrackNumber         int    `json:"trackNumber"`
	ComposerName        string `json:"composerName"`
}

type AlbumAttributes struct {
	ArtistName          string   `json:"artistName"`
	IsSingle            bool     `json:"isSingle"`
	IsComplete          bool     `json:"isComplete"`
	GenreNames          []string `json:"genreNames"`
	TrackCount          int      `json:"trackCount"`
	IsMasteredForItunes bool     `json:"isMasteredForItunes"`
	ReleaseDate         string   `json:"releaseDate"`
	Name                string   `json:"name"`
	RecordLabel         string   `json:"recordLabel"`
	Upc                 string   `json:"upc"`
	Copyright           string   `json:"copyright"`
	IsCompilation       bool     `json:"isCompilation"`
}

type SongData struct {
	ID            string         `json:"id"`
	Attributes    SongAttributes `json:"attributes"`
	Relationships struct {
		Albums struct {
			Data []struct {
				ID         string          `json:"id"`
				Type       string          `json:"type"`
				Href       string          `json:"href"`
				Attributes AlbumAttributes `json:"attributes"`
			} `json:"data"`
		} `json:"albums"`
		Artists struct {
			Href string `json:"href"`
			Data []struct {
				ID   string `json:"id"`
				Type string `json:"type"`
				Href string `json:"href"`
			} `json:"data"`
		} `json:"artists"`
	} `json:"relationships"`
}

type SongResult struct {
	Artwork struct {
		Width                int    `json:"width"`
		URL                  string `json:"url"`
		Height               int    `json:"height"`
		TextColor3           string `json:"textColor3"`
		TextColor2           string `json:"textColor2"`
		TextColor4           string `json:"textColor4"`
		HasAlpha             bool   `json:"hasAlpha"`
		TextColor1           string `json:"textColor1"`
		BgColor              string `json:"bgColor"`
		HasP3                bool   `json:"hasP3"`
		SupportsLayeredImage bool   `json:"supportsLayeredImage"`
	} `json:"artwork"`
	ArtistName             string   `json:"artistName"`
	CollectionID           string   `json:"collectionId"`
	DiscNumber             int      `json:"discNumber"`
	GenreNames             []string `json:"genreNames"`
	ID                     string   `json:"id"`
	DurationInMillis       int      `json:"durationInMillis"`
	ReleaseDate            string   `json:"releaseDate"`
	ContentRatingsBySystem struct {
	} `json:"contentRatingsBySystem"`
	Name     string `json:"name"`
	Composer struct {
		Name string `json:"name"`
		URL  string `json:"url"`
	} `json:"composer"`
	EditorialArtwork struct {
	} `json:"editorialArtwork"`
	CollectionName string `json:"collectionName"`
	AssetUrls      struct {
		Plus             string `json:"plus"`
		Lightweight      string `json:"lightweight"`
		SuperLightweight string `json:"superLightweight"`
		LightweightPlus  string `json:"lightweightPlus"`
		EnhancedHls      string `json:"enhancedHls"`
	} `json:"assetUrls"`
	AudioTraits []string `json:"audioTraits"`
	Kind        string   `json:"kind"`
	Copyright   string   `json:"copyright"`
	ArtistID    string   `json:"artistId"`
	Genres      []struct {
		GenreID   string `json:"genreId"`
		Name      string `json:"name"`
		URL       string `json:"url"`
		MediaType string `json:"mediaType"`
	} `json:"genres"`
	TrackNumber int    `json:"trackNumber"`
	AudioLocale string `json:"audioLocale"`
	Offers      []struct {
		ActionText struct {
			Short       string `json:"short"`
			Medium      string `json:"medium"`
			Long        string `json:"long"`
			Downloaded  string `json:"downloaded"`
			Downloading string `json:"downloading"`
		} `json:"actionText"`
		Type           string  `json:"type"`
		PriceFormatted string  `json:"priceFormatted"`
		Price          float64 `json:"price"`
		BuyParams      string  `json:"buyParams"`
		Variant        string  `json:"variant,omitempty"`
		Assets         []struct {
			Flavor  string `json:"flavor"`
			Preview struct {
				Duration int    `json:"duration"`
				URL      string `json:"url"`
			} `json:"preview"`
			Size     int `json:"size"`
			Duration int `json:"duration"`
		} `json:"assets"`
	} `json:"offers"`
}

type Meta struct {
	Context     string `json:"@context"`
	Type        string `json:"@type"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Tracks      []struct {
		Type  string `json:"@type"`
		Name  string `json:"name"`
		Audio struct {
			Type string `json:"@type"`
		} `json:"audio"`
		Offers struct {
			Type     string `json:"@type"`
			Category string `json:"category"`
			Price    int    `json:"price"`
		} `json:"offers"`
		Duration string `json:"duration"`
	} `json:"tracks"`
	Citation    []interface{} `json:"citation"`
	WorkExample []struct {
		Type  string `json:"@type"`
		Name  string `json:"name"`
		URL   string `json:"url"`
		Audio struct {
			Type string `json:"@type"`
		} `json:"audio"`
		Offers struct {
			Type     string `json:"@type"`
			Category string `json:"category"`
			Price    int    `json:"price"`
		} `json:"offers"`
		Duration string `json:"duration"`
	} `json:"workExample"`
	Genre         []string  `json:"genre"`
	DatePublished time.Time `json:"datePublished"`
	ByArtist      struct {
		Type string `json:"@type"`
		URL  string `json:"url"`
		Name string `json:"name"`
	} `json:"byArtist"`
}

type AutoGenerated struct {
	Data []struct {
		ID         string `json:"id"`
		Type       string `json:"type"`
		Href       string `json:"href"`
		Attributes struct {
			Artwork struct {
				Width      int    `json:"width"`
				Height     int    `json:"height"`
				URL        string `json:"url"`
				BgColor    string `json:"bgColor"`
				TextColor1 string `json:"textColor1"`
				TextColor2 string `json:"textColor2"`
				TextColor3 string `json:"textColor3"`
				TextColor4 string `json:"textColor4"`
			} `json:"artwork"`
			ArtistName          string   `json:"artistName"`
			IsSingle            bool     `json:"isSingle"`
			URL                 string   `json:"url"`
			IsComplete          bool     `json:"isComplete"`
			GenreNames          []string `json:"genreNames"`
			TrackCount          int      `json:"trackCount"`
			IsMasteredForItunes bool     `json:"isMasteredForItunes"`
			ReleaseDate         string   `json:"releaseDate"`
			Name                string   `json:"name"`
			RecordLabel         string   `json:"recordLabel"`
			Upc                 string   `json:"upc"`
			AudioTraits         []string `json:"audioTraits"`
			Copyright           string   `json:"copyright"`
			PlayParams          struct {
				ID   string `json:"id"`
				Kind string `json:"kind"`
			} `json:"playParams"`
			IsCompilation bool `json:"isCompilation"`
		} `json:"attributes"`
		Relationships struct {
			RecordLabels struct {
				Href string        `json:"href"`
				Data []interface{} `json:"data"`
			} `json:"record-labels"`
			Artists struct {
				Href string `json:"href"`
				Data []struct {
					ID         string `json:"id"`
					Type       string `json:"type"`
					Href       string `json:"href"`
					Attributes struct {
						Name string `json:"name"`
					} `json:"attributes"`
				} `json:"data"`
			} `json:"artists"`
			Tracks struct {
				Href string `json:"href"`
				Data []struct {
					ID         string `json:"id"`
					Type       string `json:"type"`
					Href       string `json:"href"`
					Attributes struct {
						Previews []struct {
							URL string `json:"url"`
						} `json:"previews"`
						Artwork struct {
							Width      int    `json:"width"`
							Height     int    `json:"height"`
							URL        string `json:"url"`
							BgColor    string `json:"bgColor"`
							TextColor1 string `json:"textColor1"`
							TextColor2 string `json:"textColor2"`
							TextColor3 string `json:"textColor3"`
							TextColor4 string `json:"textColor4"`
						} `json:"artwork"`
						ArtistName          string   `json:"artistName"`
						URL                 string   `json:"url"`
						DiscNumber          int      `json:"discNumber"`
						GenreNames          []string `json:"genreNames"`
						HasTimeSyncedLyrics bool     `json:"hasTimeSyncedLyrics"`
						IsMasteredForItunes bool     `json:"isMasteredForItunes"`
						DurationInMillis    int      `json:"durationInMillis"`
						ReleaseDate         string   `json:"releaseDate"`
						Name                string   `json:"name"`
						Isrc                string   `json:"isrc"`
						AudioTraits         []string `json:"audioTraits"`
						HasLyrics           bool     `json:"hasLyrics"`
						AlbumName           string   `json:"albumName"`
						PlayParams          struct {
							ID   string `json:"id"`
							Kind string `json:"kind"`
						} `json:"playParams"`
						TrackNumber  int    `json:"trackNumber"`
						AudioLocale  string `json:"audioLocale"`
						ComposerName string `json:"composerName"`
					} `json:"attributes"`
					Relationships struct {
						Artists struct {
							Href string `json:"href"`
							Data []struct {
								ID         string `json:"id"`
								Type       string `json:"type"`
								Href       string `json:"href"`
								Attributes struct {
									Name string `json:"name"`
								} `json:"attributes"`
							} `json:"data"`
						} `json:"artists"`
					} `json:"relationships"`
				} `json:"data"`
			} `json:"tracks"`
		} `json:"relationships"`
	} `json:"data"`
}

type Alac struct {
	mp4.FullBox `mp4:"extend"`

	FrameLength       uint32 `mp4:"size=32"`
	CompatibleVersion uint8  `mp4:"size=8"`
	BitDepth          uint8  `mp4:"size=8"`
	Pb                uint8  `mp4:"size=8"`
	Mb                uint8  `mp4:"size=8"`
	Kb                uint8  `mp4:"size=8"`
	NumChannels       uint8  `mp4:"size=8"`
	MaxRun            uint16 `mp4:"size=16"`
	MaxFrameBytes     uint32 `mp4:"size=32"`
	AvgBitRate        uint32 `mp4:"size=32"`
	SampleRate        uint32 `mp4:"size=32"`
}

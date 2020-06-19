package app

import (
	"fmt"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/gocolly/colly"
)

const (
	streamIDClassPrefix         = "belongs_to_"
	streamTableBodySelector     = ".streams_table"
	streamMainRowSelector       = "tr.border-solid"
	streamLinkDataAttr          = "data-clipboard-text"
	streamLinkDataSelector      = "." + streamIDClassPrefix + "%s span[" + streamLinkDataAttr + "]"
	streamCountryDataAttr       = "title"
	streamCountryDataSelector   = "td.flag > a > img[" + streamCountryDataAttr + "]"
	streamChannelDataSelector   = "td > span.channel_name"
	streamLivelinessSelector    = "td > div.live > div.live"
	streamStatusDataAttr        = "title"
	streamStatusSelector        = "td > div.state[" + streamStatusDataAttr + "]"
	streamLastCheckedSelector   = "td.channel_checked > span"
	streamLastCheckedDateFormat = "02 Jan 2006"
	streamHDFormatSelector      = "td:nth-child(6)"
	streamMbpsSelector          = "td:nth-child(7) > span"
)

// GetStreamTableSelector returns the stream table querySelector
func GetStreamTableSelector() string {
	return streamTableBodySelector
}

// Stream represents a channel stream
type Stream struct {
	ID          string   `json:"id"`
	Channel     string   `json:"channel"`
	Link        string   `json:"link"`
	Country     string   `json:"country"`
	Liveliness  string   `json:"liveliness"`
	Status      string   `json:"status"`
	LastChecked string   `json:"lastChecked"`
	Format      string   `json:"format"`
	Mbps        string   `json:"mbps"`
	URI         []string `json:"URI"`
}

type streamsData struct {
	All       []*Stream
	ByID      map[string]*Stream
	ByCountry map[string][]*Stream
}

// Streams contains all found streams during scraping
var Streams = &streamsData{
	All:       []*Stream{},
	ByID:      map[string]*Stream{},
	ByCountry: map[string][]*Stream{},
}

func getClassList(node *goquery.Selection) (classList []string) {
	classString, _ := node.Attr("class")

	return strings.Split(classString, " ")
}

func getStreamIDFromClassList(classList []string) (streamID string, className string) {
	for _, className := range classList {
		if strings.Contains(className, streamIDClassPrefix) {
			return strings.Replace(className, streamIDClassPrefix, "", 1), className
		}
	}
	return "", ""
}

func getStreamID(node *goquery.Selection) (streamID string, className string) {
	return getStreamIDFromClassList(getClassList(node))
}

func getStream(streamTable *colly.HTMLElement, streamDataRow *goquery.Selection) *Stream {

	streamID, _ := getStreamID(streamDataRow)
	streamLinkSelector := fmt.Sprintf(streamLinkDataSelector, streamID)
	streamLink, _ := streamTable.DOM.Find(streamLinkSelector).Attr(streamLinkDataAttr)
	streamCountry, _ := streamDataRow.Find(streamCountryDataSelector).Attr(streamCountryDataAttr)
	streamChannel := streamDataRow.Find(streamChannelDataSelector).Text()
	streamLiveliness := streamDataRow.Find(streamLivelinessSelector).Text()
	streamStatus, _ := streamDataRow.Find(streamStatusSelector).Attr(streamStatusDataAttr)
	streamLastChecked := streamDataRow.Find(streamLastCheckedSelector).Text()
	streamLastCheckedTime, _ := time.Parse(streamLastCheckedDateFormat, streamLastChecked)
	if !streamLastCheckedTime.IsZero() {
		streamLastChecked = streamLastCheckedTime.Format(time.RFC3339)
	}
	streamHDFormat := streamDataRow.Find(streamHDFormatSelector).Text()
	if streamHDFormat == "" {
		streamHDFormat = "SD"
	}
	streamMbpsSelector := streamDataRow.Find(streamMbpsSelector).Text()

	return &Stream{
		ID:          streamID,
		Link:        streamLink,
		Channel:     streamChannel,
		Country:     strings.ToLower(streamCountry),
		Liveliness:  streamLiveliness,
		Status:      strings.ToLower(streamStatus),
		Format:      strings.ToLower(streamHDFormat),
		Mbps:        streamMbpsSelector,
		LastChecked: streamLastChecked,
	}
}

func getStreamURIBase(urlPath string) string {
	return strings.Split(urlPath, "/")[1]
}

func uniqueSlice(URIs []string) []string {
	uniq := []string{}
	seen := map[string]bool{}
	for _, str := range URIs {
		if seen[str] {
			continue
		}
		seen[str] = true
		uniq = append(uniq, str)
	}
	return uniq
}

// HandleStreamTable registers the stream table scraper
func HandleStreamTable(c *colly.Collector) func(el *colly.HTMLElement) {

	return func(streamTable *colly.HTMLElement) {
		el := streamTable.DOM.Find(streamMainRowSelector)

		for i := range el.Nodes {
			streamDataRow := el.Eq(i)
			stream := getStream(streamTable, streamDataRow)
			if stream.ID == "" {
				continue
			}

			existingStream := Streams.ByID[stream.ID]
			hasSeen := existingStream != nil
			if hasSeen {
				stream = existingStream
			} else {
				Streams.ByID[stream.ID] = stream
			}

			urlPath := getStreamURIBase(streamTable.Request.URL.Path)
			stream.URI = uniqueSlice(append(stream.URI, urlPath))
			Streams.All = append(Streams.All, stream)
			if Streams.ByCountry[stream.Country] == nil {
				Streams.ByCountry[stream.Country] = []*Stream{}
			}
			Streams.ByCountry[stream.Country] = append(Streams.ByCountry[stream.Country], stream)

		}
	}
}

// HandleFollowLinks will follow all anchor tags with hrefs
func HandleFollowLinks(c *colly.Collector) func(el *colly.HTMLElement) {
	return func(el *colly.HTMLElement) {
		link := el.Attr("href")
		if link == "" {
			return
		}

		isM3U := strings.Contains(link, ".m3u8")
		if isM3U {
			return
		}

		c.Visit(el.Request.AbsoluteURL(link))
	}
}

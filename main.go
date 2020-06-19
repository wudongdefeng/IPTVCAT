package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"

	app "iptvcat-scraper/pkg"

	"github.com/gocolly/colly"
)

const iptvCatDomain = "iptvcat.com"
const iptvCatURL = "https://" + iptvCatDomain
const aHref = "a[href]"

func writeToFile() {
	streamsAll, err := json.Marshal(app.Streams.All)
	streamsCountry, err := json.Marshal(app.Streams.ByCountry)
	if err != nil {
		fmt.Println("error:", err)
	}
	ioutil.WriteFile("data/all-streams.json", streamsAll, 0644)
	ioutil.WriteFile("data/all-by-country.json", streamsCountry, 0644)
	for key, val := range app.Streams.ByCountry {
		streamsCountry, err := json.Marshal(val)
		if err != nil {
			fmt.Println("error:", err)
		}
		ioutil.WriteFile("data/countries/"+key+".json", streamsCountry, 0644)
	}
}

func main() {
	c := colly.NewCollector(
		colly.AllowedDomains(iptvCatDomain),
	)

	c.OnResponse(func(r *colly.Response) {
		fmt.Println("Visited", r.Request.URL)
	})

	c.OnHTML(aHref, app.HandleFollowLinks(c))
	c.OnHTML(app.GetStreamTableSelector(), app.HandleStreamTable(c))

	c.OnScraped(func(r *colly.Response) {
		fmt.Println("Finished", r.Request.URL)
	})

	c.OnError(func(r *colly.Response, err error) {
		fmt.Printf("Error: %d %s\n", r.StatusCode, r.Request.URL)
	})

	c.Visit(iptvCatURL)
	c.Wait()
	writeToFile()
}

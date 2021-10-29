package japan

import (
	_ "errors"
	"fmt"
	"io"
	_ "io/ioutil"
	"log"
	_ "net/http"
	_ "strings"

	"github.com/NagoDede/notamloader/webclient"
	"github.com/PuerkitoBio/goquery"
)

type JpNotamMapSubmit struct {
	Enroute string  `json:"enroute"`
	Period string `json:"period"`
	DispScopeE string `json:"dispScopeE"`
	DispScopeW string `json:"dispScopeW"`
}


func (mapSearch *JpNotamMapSubmit) ListNotamReferences(httpClient *webclient.AisWebClient, webPage string, nextWebPage string) []JpNotamDispForm {
	urlValues := webclient.StructToMap(mapSearch)
	httpClient.RLock()
	
	resp, err := httpClient.Client.PostForm(webPage, urlValues)
	httpClient.RUnlock()
	if err != nil {
		log.Println("Unable to perform mapsearch")
		log.Fatal(err)
	}
	defer resp.Body.Close()

	return listEnrouteNotams(httpClient, resp.Body, nextWebPage)
}

func listEnrouteNotams(httpClient *webclient.AisWebClient, body io.ReadCloser, nextWebPage string) []JpNotamDispForm {
	var notamrefs = make([]JpNotamDispForm, 0)
	var page int
	page = 1
	return subListEnrouteNotams(httpClient, body, notamrefs[:], nextWebPage, &page)
}

// Extract the data from the downloaded webpage.
func subListEnrouteNotams(httpClient *webclient.AisWebClient, body io.ReadCloser, notamRefs []JpNotamDispForm, nextWebPage string, page *int) []JpNotamDispForm {
	doc, err := goquery.NewDocumentFromReader(body)
	if err != nil {
		fmt.Println("Unable to extract data from en Route Webpage (subListEnrouteNotams)")
		log.Fatal(err)
	}

	doc.Find(`a[href*="javascript:formSubmit"]`).Each(
		func(index int, a *goquery.Selection) {
			href, _ := a.Attr("href")
			data, err := extractFormData(href)
			if (err == nil){
				notamRefs = append(notamRefs, *data)
			} else {
				fmt.Printf("!!! Error: %s \n", err)
			}
		})

	if doc.Find(`a[onclick*="javascript:postLink('next')"]`).Length() > 0 {
		*page = *page + 1
		fmt.Printf("Page %d \n", *page)
		var anchor JpNotamAnchor
		anchor.anchor = `next`
		urlAnchor :=  webclient.StructToMap(&anchor)
		httpClient.RLock()
		resp, err := httpClient.Client.PostForm(nextWebPage, urlAnchor)
		httpClient.RUnlock()
		if err == nil {
			defer resp.Body.Close()
			return subListNotams(httpClient, resp.Body, notamRefs[:], nextWebPage, page)
		} else 	{
			log.Printf("Error to recover next page: %d", *page)
			log.Println(err)
		} 
	}
return notamRefs
}
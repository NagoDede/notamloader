package japan

import (
	"fmt"
	"log"
	"net/http"
	"net/url"

	"github.com/PuerkitoBio/goquery"
)

type WebConfig struct {
	CountryDir      string `json:"country"`
	LoginPage       string `json:"loginPage"`
	NotamFirstPage  string `json:"notamFirstPage"`
	NotamDetailPage string `json:"notamDetailPage"`
	NotamNextPage   string `json:"notamNextPage"`
	MapPage         string `json:"mapPage"`
	MapAnswerPage   string `json:"mapAnswerPage"`
	LogOutPage      string `json:"logoutpage"`
	//httpClient      http.Client //share the httpClient
	IsConnected bool
}

//Values used to generate a Next webpage request
var nextFormData = url.Values{
	"anchor": {"next"},
}

func (web *WebConfig) GetFirstPage(httpClient http.Client, values url.Values) *goquery.Document {
	return web.GetPage(httpClient, web.NotamFirstPage, values)
}

func (web *WebConfig) GetNextPage(httpClient http.Client) *goquery.Document {
	return web.GetPage(httpClient, web.NotamNextPage, nextFormData)
}

func (web *WebConfig) GetPage(httpClient http.Client, url string, values url.Values) *goquery.Document {
	resp, err := httpClient.PostForm(url, values)
	if err != nil {
		log.Println("If error due to certificate problem, install ca-certificates")
		log.Fatal(err)
	}
	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		fmt.Println("Unable to extract data from Webpage (GetPage)")
		log.Fatal(err)
	}
	return doc
}

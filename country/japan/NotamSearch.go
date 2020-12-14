package japan

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

type JpNotamSearchForm struct {
	location   string `json:"location"`
	notamKbn   string `json:"notamKbn"`
	selLoc     string `json:"selLoc"`
	periodFrom string `json:"periodFrom"`
	periodTo   string `json:"periodTo"`
	period     string `json:"period"`
	notamCode  string `json:"notamCode"`
	dispScopeA string `json:"dispScopeA"`
	dispScopeE string `json:"dispScopeE"`
	dispScopeW string `json:"dispScopeW"`
	firstFlg   string `json:"firstFlg"`
	lower      string `json:"lower"`
	upper      string `json:"upper"`
	itemE      string `json:"itemE"`
}

var pages = 1              //use to count the pages (display only)
var httpClient http.Client //share the httpClient
var jpData *JpData         //store the japan website data

//Values used to generate a Next webpage request
var nextFormData = url.Values{
	"anchor": {"next"},
}

//ToUrlValues converts the structure to an url.Values structure.
func (nsf JpNotamSearchForm) ToUrlValues() (values url.Values) {
	return structToUrlValues(&nsf)
}

/* ListNotamReferences retrieves and lists the Notams identified on the Japan Notam website.
If Notams are printed on several pages, it retrieves the Notams on all the pages.
httpClient is the client initialized to the Japan Notam website.
*/
func (nsf JpNotamSearchForm) ListNotamReferences(httpClient http.Client, jpd *JpData) []*JpNotamDispForm {

	jpData = jpd
	pages = 1

	var notamRefs []*JpNotamDispForm //will contain the results

	//Send the request to the NotamFirstPage
	resp, err := httpClient.PostForm(jpd.WebConfig.NotamFirstPage, nsf.ToUrlValues())
	if err != nil {
		log.Println("If error due to certificate problem, install ca-certificates")
		log.Fatal(err)
	}
	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		fmt.Println("No url found for navaid extraction")
		log.Fatal(err)
	}
	notamRefs = listNotams(doc, notamRefs, httpClient)
	return notamRefs
}

/* 	listNotams lists the notam on the current webpage.
If there is several webpages, run accross them by using the getNextPages function.
*/
func listNotams(doc *goquery.Document, notamRefs []*JpNotamDispForm, httpClient http.Client) []*JpNotamDispForm {
	//read the current webpages
	doc.Find(`a[href*="javascript:formSubmit"]`).Each(
		func(index int, a *goquery.Selection) {
			href, _ := a.Attr("href")

			notamRefs = append(notamRefs, extractFormData(href))
		})

	return getNextPages(doc, notamRefs, httpClient)
}

/**
getNextPages identifies if there is other pages. If yes, it sends the request to get the next page.
It is an iterative function, throught the listNotams function.
*/
func getNextPages(doc *goquery.Document, notamRefs []*JpNotamDispForm, httpClient http.Client) []*JpNotamDispForm {

	thereIsNext := len(doc.Find(`a[onclick="javascript:postLink('next')"]`).Nodes) > 0

	if thereIsNext {

		pages++
		fmt.Printf("Go to page %d \n", pages)

		resp, err := httpClient.PostForm(jpData.NotamNextPage, nextFormData)
		if err != nil {
			log.Println("If error due to certificate problem, install ca-certificates")
			log.Fatal(err)
		}
		defer resp.Body.Close()

		doc, err := goquery.NewDocumentFromReader(resp.Body)
		if err != nil {
			fmt.Println("No url found for navaid extraction")
			log.Fatal(err)
		}
		notamRefs = listNotams(doc, notamRefs, httpClient)
	}
	return notamRefs
}

func extractFormData(jsref string) *JpNotamDispForm {
	ns := strings.Replace(jsref, "javascript:formSubmit(", "", -1)
	ns = strings.Replace(ns, ")", "", -1)
	splitted := strings.Split(ns, ",")

	notamInfo := new(JpNotamDispForm)
	notamInfo.location = strings.TrimSpace(splitted[0])
	notamInfo.location = strings.Trim(notamInfo.location, "'")

	notamInfo.notam_series = strings.TrimSpace(splitted[1])
	notamInfo.notam_series = strings.Trim(notamInfo.notam_series, "'")

	notamInfo.notam_no = strings.TrimSpace(splitted[2])
	notamInfo.notam_no = strings.Trim(notamInfo.notam_no, "'")

	notamInfo.notam_year = strings.TrimSpace(splitted[3])
	notamInfo.notam_year = strings.Trim(notamInfo.notam_year, "'")

	notamInfo.anchor = strings.TrimSpace(splitted[4])
	notamInfo.anchor = strings.Trim(notamInfo.anchor, "'")

	notamInfo.dispOrder = strings.TrimSpace(splitted[5])
	notamInfo.dispOrder = strings.Trim(notamInfo.dispOrder, "'")

	notamInfo.notamKbn = strings.TrimSpace(splitted[6])
	notamInfo.notamKbn = strings.Trim(notamInfo.notamKbn, "'")

	notamInfo.dispFromTime = strings.TrimSpace(splitted[7])
	notamInfo.dispFromTime = strings.Trim(notamInfo.dispFromTime, "'")

	return notamInfo

}

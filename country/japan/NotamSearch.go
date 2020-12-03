package japan

import (
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"io"
	"log"
	"net/http"
	"strings"
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

func (nsf *JpNotamSearchForm) ListNotamReferences(httpClient http.Client, webpage string) []*JpNotamDispForm {
	urlValues := structToMap(nsf)
	//connect to the website
	resp, err := httpClient.PostForm(webpage, urlValues)
	if err != nil {
		log.Println("If error due to certificate problem, install ca-certificates")
		log.Fatal(err)
	}

	return listNotams(resp.Body)
}

func listNotams(body io.ReadCloser) []*JpNotamDispForm {

	doc, err := goquery.NewDocumentFromReader(body)
	if err != nil {
		fmt.Println("No url found for navaid extraction")
		log.Fatal(err)
	}

	var notamRefs []*JpNotamDispForm

	doc.Find(`a[href*="javascript:formSubmit"]`).Each(
		func(index int, a *goquery.Selection) {
			href, _ := a.Attr("href")

			notamRefs = append(notamRefs, extractFormData(href))
		})
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

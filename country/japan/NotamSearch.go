package japan

import (
	"errors"
	"fmt"
	"io"
	"log"
	_ "net/http"
	"strings"

	"github.com/NagoDede/notamloader/webclient"
	"github.com/PuerkitoBio/goquery"
)

//Structure used to submit data for airport / keycode search
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

//structure used to submit a map search
// if location empty, retrieve all enroute data
type JpNotamAnchor struct {
	anchor string
}

func (nsf *JpNotamSearchForm) ListNotamReferences(httpClient *webclient.AisWebClient, webpage string, nextWebPage string) []JpNotamDispForm {
	urlValues := structToMap(nsf)

	//connect to the website
	httpClient.RLock()
	resp, err := httpClient.Client.PostForm(webpage, urlValues)
	httpClient.RUnlock()
	if err != nil {
		log.Printf("Unable to submit form for %v \n", nsf)
		log.Fatal(err)
	}
	defer resp.Body.Close()
	return listNotams(httpClient, resp.Body, nextWebPage)
}

func listNotams(httpClient *webclient.AisWebClient, body io.ReadCloser, nextWebPage string) []JpNotamDispForm {
	var notamrefs = make([]JpNotamDispForm, 0)
	var page int
	page = 1
	return subListNotams(httpClient, body, notamrefs[:], nextWebPage, &page)
}

// Extract the data from the downloaded webpage.
func subListNotams(httpClient *webclient.AisWebClient, body io.ReadCloser, notamRefs []JpNotamDispForm, nextWebPage string, page *int) []JpNotamDispForm {
	doc, err := goquery.NewDocumentFromReader(body)
	if err != nil {
		fmt.Println("No url found for navaid extraction")
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
		urlAnchor := structToMap(&anchor)
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

// Extract relevant notam data from a specific string.
// Return an error if the location OR notam_no OR notam_year is empty.
// Relevant data is defined in a javascript command like this one:
// "javascript:formSubmit('RJAF', ' ', '326','21','1','1','1','21/10/22 11:43')"
// The function is defined as is:
// function formSubmit(alpha1,alpha2,alpha3,alpha4,alpha5,alpha6,alpha7,alpha8) {
// 	document.notamInfo.location.value = alpha1;
// 	document.notamInfo.notam_series.value = alpha2;
// 	document.notamInfo.notam_no.value = alpha3;
// 	document.notamInfo.notam_year.value = alpha4;
// 	document.notamInfo.anchor.value = alpha5;
// 	document.notamInfo.dispOrder.value = alpha6;
// 	document.notamInfo.notamKbn.value = alpha7;
// 	document.notamInfo.dispFromTime.value = alpha8;
// 	....
func extractFormData(jsref string) (*JpNotamDispForm, error) {
	ns := strings.Replace(jsref, "javascript:formSubmit(", "", -1)
	ns = strings.Replace(ns, ")", "", -1)
	splitted := strings.Split(ns, ",")

	if (ns == "") {
		return nil, errors.New("Empty text identified in " + jsref)
	}

	notamInfo := new(JpNotamDispForm)
	notamInfo.location = strings.TrimSpace(splitted[0])
	notamInfo.location = strings.Trim(notamInfo.location, "'")
	if (notamInfo.location == "") {
		return nil, errors.New("No Location identified in " + jsref)
	}

	notamInfo.notam_series = strings.TrimSpace(splitted[1])
	notamInfo.notam_series = strings.Trim(notamInfo.notam_series, "'")

	notamInfo.notam_no = strings.TrimSpace(splitted[2])
	notamInfo.notam_no = strings.Trim(notamInfo.notam_no, "'")
	if (notamInfo.notam_no == "") {
		return nil, errors.New("No notam number identified in " + jsref)
	}

	notamInfo.notam_year = strings.TrimSpace(splitted[3])
	notamInfo.notam_year = strings.Trim(notamInfo.notam_year, "'")
	if (notamInfo.notam_year == "") {
		return nil, errors.New("No year identified in " + jsref)
	}

	notamInfo.anchor = strings.TrimSpace(splitted[4])
	notamInfo.anchor = strings.Trim(notamInfo.anchor, "'")

	notamInfo.dispOrder = strings.TrimSpace(splitted[5])
	notamInfo.dispOrder = strings.Trim(notamInfo.dispOrder, "'")

	notamInfo.notamKbn = strings.TrimSpace(splitted[6])
	notamInfo.notamKbn = strings.Trim(notamInfo.notamKbn, "'")

	notamInfo.dispFromTime = strings.TrimSpace(splitted[7])
	notamInfo.dispFromTime = strings.Trim(notamInfo.dispFromTime, "'")

	return notamInfo, nil

}

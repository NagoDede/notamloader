package asecna

import (
	"fmt"
	"io"

	"strconv"
	"strings"

	"log"
	"net/http"
	"net/url"

	_ "strconv"
	"time"

	"github.com/NagoDede/notamloader/notam"
	_ "github.com/NagoDede/notamloader/webclient"
	"github.com/PuerkitoBio/goquery"
)

func (def *DefData) RetrieveAllNotams(afs string) *NotamList {
	notamList := NewNotamList()

	//allNotams := notamList.notamList//[]*FranceNotam{}
	for _, icaoCode := range def.RequiredLocations[afs] {

		nbNotams, txtNotams := def.performRequest(icaoCode)
		fmt.Printf("Expected %d Notams - Retrieved Notam Text: %d \n", nbNotams, len(txtNotams))
		notamList = def.createNotamsFromText(afs, txtNotams, notamList)
	}
	return notamList
}

func (def *DefData) performRequest(firCode string) (int, []string) {
	form := NewFormRequest(firCode)
	resp, err := def.SendRequest(form.Encode())
	if err != nil {
		log.Fatalln(err)
	}
	defer resp.Body.Close()
	//   b, err := io.ReadAll(resp.Body)
	//   if err != nil {
	//   	log.Fatalln(err)
	//   }
	//  fmt.Println(string(b))
	//nbNotams := 0 //extractNumberOfNotams(resp.Body)
	ntmTxt := &[]string{}
	nbNotams, ntmTxt := extractNotams(resp.Body, ntmTxt)

	count := 1
	for len(*ntmTxt) < nbNotams {
		url := fmt.Sprintf("%s?from=%d", def.NotamRequestUrl, count*form.Maxrows)
		resp2, err := def.SendRequestToUrl(url)

		if err != nil {
			log.Fatalln(err)
		}
		defer resp2.Body.Close()
		nbNotams, ntmTxt = extractNotams(resp2.Body, ntmTxt)
		count = count + 1
		fmt.Printf("Page %d Proceed \n", count)
	}

	fmt.Printf("Expected NOTAM for %s : %d %d \n", firCode, nbNotams, len(*ntmTxt))
	return nbNotams, *ntmTxt
}

func (def *DefData) createNotamsFromText(afs string, notamsText []string, allNotams *NotamList) *NotamList {

	for _, ntmTxt := range notamsText {
		ntm := NewNotam(afs)
		ntm.NotamAdvanced = notam.FillNotamFromText(ntm.NotamAdvanced, ntmTxt)
		//ntm.Id = ntm.GetKey()
		_, ok := allNotams.notamList[ntm.Id]
		if !ok {
			allNotams.notamList[ntm.Id] = ntm
		}
		//allNotams = append(allNotams, ntm)
	}
	return allNotams
}

func extractNotams(body io.ReadCloser, sNotams *[]string) (int, *[]string) {

	doc, err := goquery.NewDocumentFromReader(body)
	if err != nil {
		fmt.Println("Unable to retrieve NOTAM info (see extractNotams)")
		log.Fatal(err)
	}

	var nbNotam = 0
	doc.Find(`p[id*="result"]`).Each(
		func(index int, a *goquery.Selection) {
			txt := a.Text()
			if strings.Index(txt, "sur") < 0 || strings.Index(txt, "réponse") < 0 {
				nbNotam = 0
			} else {
				txt = txt[strings.Index(txt, "sur")+4 : strings.Index(txt, "réponse")]
				txt = strings.Trim(txt, " ")
				nbNotam, err = strconv.Atoi(txt)
				if err != nil {
					nbNotam = 0
				}
			}
		})

	doc.Find(`div[id*="notam"]`).Each(
		func(index int, a *goquery.Selection) {

			txt := a.Text()
			if strings.Index(txt, "(") < 0 {
				*sNotams = append(*sNotams, txt)
			} else {
				txt = txt[strings.Index(txt, "(")+1:]
				txt = strings.Trim(txt, " \n\r\t")
				txt = txt[:len(txt)-1]
				*sNotams = append(*sNotams, txt)
			}
		})

	return nbNotam, sNotams
}

func extractNumberOfNotams(body io.ReadCloser) int {
	var nbNotam = 0
	doc, err := goquery.NewDocumentFromReader(body)
	if err != nil {
		fmt.Println("Unable to retrieve NOTAM info (see extractNotams)")
		log.Fatal(err)
	}

	doc.Find(`p[id*="result"]`).Each(
		func(index int, a *goquery.Selection) {
			txt := a.Text()
			txt = txt[strings.Index(txt, "sur")+4 : strings.Index(txt, "réponse")]
			txt = strings.Trim(txt, " ")
			nbNotam, _ = strconv.Atoi(txt)
		})

	return nbNotam
}

func (def *DefData) SendRequest(form url.Values) (*http.Response, error) {
	return aisClient.Client.PostForm(def.NotamRequestUrl, form) //aisClient.SendPost(def.NotamRequestUrl, form)
}

func (def *DefData) SendRequestToUrl(toUrl string) (*http.Response, error) {
	return aisClient.Client.Get(toUrl) //aisClient.SendPost(def.NotamRequestUrl, form)
}
func getFormatedDate() string {
	currentDate := time.Now().UTC()
	return currentDate.Format("2006/01/02")
}

func getFormatedHour() string {
	currentUtc := time.Now().UTC().Add(10 * time.Minute)
	return currentUtc.Format("15:04")

}

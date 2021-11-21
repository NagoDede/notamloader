package france

import (
	"fmt"
	"math"
	"regexp"
	"strings"

	"io"
	"log"
	"net/http"
	"net/url"
	"strconv"
	_ "strconv"
	"time"

	"github.com/NagoDede/notamloader/notam"
	_ "github.com/NagoDede/notamloader/webclient"
	"github.com/PuerkitoBio/goquery"
)

func (def *DefData) RetrieveAllNotams(afs string) *FranceNotamList {
	notamList := NewFranceNotamList()

	//allNotams := notamList.notamList//[]*FranceNotam{}
	for _, icaoCode := range def.RequiredFirLocations[afs] {
		//There is a server limitation, above 200/300 Notams, webpage is to big and server cannot handle it.
		//So, in a first step, we request the resume (only notam ID and title)
		//and in a second step, we perform complete request by batch process.
		//This capabilities is ensured by the fact we use the same form for the request.
		//In the complete request, the form is updated with the reference of the requested NOTAMS
		nbNotams, form := def.performResumeRequest(icaoCode)
		notamList = def.performCompleteRequest(afs, nbNotams, form, notamList)
	}
	return notamList
}

func (def *DefData) performResumeRequest(icaoCode string) (int, *FormRequest) {
	form := NewFormResumeRequest(icaoCode, getFormatedDate(), getFormatedHour())
	resp, err := def.SendFirRequest(form.Encode())
	if err != nil {
		log.Fatalln(err)
	}
	defer resp.Body.Close()
	//  b, err := io.ReadAll(resp.Body)
	//  if err != nil {
	//  	log.Fatalln(err)
	//  }
	// fmt.Println(string(b))
	nbNotams := extractNumberOfNotams(resp.Body)
	fmt.Printf("Expected NOTAM for %s : %d \n", icaoCode, nbNotams)
	return nbNotams, form
}

// Retrieve all the Notams that have been previsouly identified thanks an initial form request.
// Update the allNotams list with the new Notams and return the new table.
func (def *DefData) performCompleteRequest(afs string, nbNotams int, initForm *FormRequest, allNotams *FranceNotamList) *FranceNotamList {
	const maxreq = 100
	fentier, _ := math.Modf(float64(nbNotams) / float64(maxreq))
	entier := int(fentier)
	count := 0

	// Dedicated function that requests the notams from min_id to max_id
	individualRequest := func(min_id int, cnt int) {
		resp, err := def.SendFirRequest(initForm.EncodeForComplet(min_id, cnt))
		if err != nil {
			log.Fatalln(err)
		}
		defer resp.Body.Close()
		notamsText := extractNotams(resp.Body)
		allNotams = def.createNotamsFromText(afs, notamsText, allNotams)
		count = count + len(notamsText)
		fmt.Printf("notams: %d / %d total notams: %d \n", count, nbNotams, len(allNotams.notamList))
	}

	for i := 0; i < entier; i++ {
		fmt.Printf("Request %d(%d to %d )/ %d \n", i+1, i*maxreq, (i+1)*maxreq, entier+1)
		individualRequest(i*maxreq+1, maxreq)
	}

	if (nbNotams-(entier*maxreq) >= 0) {
		fmt.Printf("Final request %d(%d to %d) / %d \n", entier+1, entier*maxreq, nbNotams, entier+1 )
	individualRequest(entier*maxreq+1, nbNotams-(entier*maxreq))
	} else {
		fmt.Printf("!!! Error - performCompleteRequest: %s, nbNotams: %d, entier: %d", afs, nbNotams, entier)
	}

	return allNotams
}

func (def *DefData)createNotamsFromText(afs string,notamsText []string, allNotams *FranceNotamList) *FranceNotamList {
	//notams := []*FranceNotam{}
	for _, ntmTxt := range notamsText {
		ntm := NewFranceNotam(afs)
		ntm.NotamAdvanced = notam.FillNotamFromText(ntm.NotamAdvanced, ntmTxt)

		_, ok :=  allNotams.notamList[ntm.Id]
		if (!ok) {
			allNotams.notamList[ntm.Id] = ntm
		}
		//allNotams = append(allNotams, ntm)
	}
	return allNotams
}

func extractNotams(body io.ReadCloser) []string {
	sNotams := []string{}

	doc, err := goquery.NewDocumentFromReader(body)
	if err != nil {
		fmt.Println("Unable to retrieve NOTAM info (see extractNotams)")
		log.Fatal(err)
	}

	doc.Find(`font[class="NOTAMBulletin"]`).Each(
		func(index int, a *goquery.Selection) {
			if len(a.Text()) > 10 {
				sNotams = append(sNotams, a.Text())
			}
		})

	return sNotams
}

func extractNumberOfNotams(body io.ReadCloser) int {
	var nbNotam = 0
	doc, err := goquery.NewDocumentFromReader(body)
	if err != nil {
		fmt.Println("Unable to retrieve NOTAM info (see extractNotams)")
		log.Fatal(err)
	}

	f := func(i int, sel *goquery.Selection) bool {
		return strings.Contains(sel.Text(), "Number of NOTAM")
	}

	doc.Find(`font[class="CorpsBulletin"]`).FilterFunction(f).Each(
		func(index int, a *goquery.Selection) {
			re := regexp.MustCompile("[0-9]+")
			nbTxt := re.FindString(a.Text())
			nbNotam, _ = strconv.Atoi(nbTxt)
		})

	return nbNotam
}

func (def *DefData) SendFirRequest(form url.Values) (*http.Response, error) {
	return aisClient.Client.PostForm(def.NotamFirRequestUrl, form) //aisClient.SendPost(def.NotamRequestUrl, form)
}

func getFormatedDate() string {
	currentDate := time.Now().UTC()
	return currentDate.Format("2006/01/02")
}

func getFormatedHour() string {
	currentUtc := time.Now().UTC().Add(10 * time.Minute)
	return currentUtc.Format("15:04")

}

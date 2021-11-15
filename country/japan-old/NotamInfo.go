package japan

import (
	"errors"
	"fmt"
	"io"
	_ "io/ioutil"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/NagoDede/notamloader/notam"
	"github.com/NagoDede/notamloader/webclient"
	"github.com/PuerkitoBio/goquery"
)

type JpNotamDispForm struct {
	location     string
	notam_series string
	notam_no     string
	notam_year   string
	anchor       string
	dispOrder    string
	notamKbn     string
	dispFromTime string
}

func (ndf *JpNotamDispForm) GetKey() string {
	return "JPN-" + ndf.location + "-" + ndf.Number()
}
func (ndf *JpNotamDispForm) Number() string {
	number, _:= strconv.Atoi(ndf.notam_no)
	year, _:= strconv.Atoi(ndf.notam_year)
	return fmt.Sprintf("%04d/%02d",number, year )
}

func (ndf *JpNotamDispForm) FillInformation(httpClient *webclient.AisWebClient, url string, countryCode string) (*notam.Notam, error) {

	urlValues :=  webclient.StructToMap(ndf)
	httpClient.RLock()
	resp, err := httpClient.Client.PostForm(url, urlValues)
	defer httpClient.RUnlock()
	
	if resp != nil {
		notam := notamText(resp.Body)
		notam.AfsCode=countryCode
		notam.Id = notam.GetKey() //.CountryCode +"/" + notam.Icaolocation + "/" +  notam.NotamReference.Number 
		defer resp.Body.Close()
		return notam, nil
	} else {
		fmt.Println("Error in FillInformation - Cannot retrieve all the data")
		fmt.Println(ndf)
		fmt.Println(err)
		return nil, errors.New("Nil answer")
	}
}

func postNotamDetail(client *webclient.AisWebClient, data url.Values, url string) (resp *http.Response, err error) {

	req, err := http.NewRequest("POST", url, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept-Encoding", "gzip;q=1.0, deflate, br, identity;q=0")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9,fr;q=0.8")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Cache-Control", "max-age=0")
//	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.9") //req.AddCookie(client.Jar.Cookies())
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9") //req.AddCookie(client.Jar.Cookies())
	req.Header.Set("Sec-Fetch-Site", "same-origin")
	req.Header.Set("Sec-Fetch-Mode", "navigate")
	req.Header.Set("Sec-Fetch-User", "?1")
	req.Header.Set("Sec-Fetch-Dest", "document")
	req.Header.Set("Upgrade-Insecuree-Requests", "1")
	req.Header.Set("Sec-GPC", "1")
	req.Header.Set("Referer", "https://aisjapan.mlit.go.jp/KeySearcherAction.do")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/95.0.4638.54 Safari/537.36")
	req.Header.Set("Host","aisjapan.mlit.go.jp")
	req.Header.Set("Origin", "https://aisjapan.mlit.go.jp")
	req.Header.Set("Referer", "https://aisjapan.mlit.go.jp/Search.do")
	client.RLock()
	defer client.RUnlock()
	res, err := client.Client.Do(req)
	return res, err
}

func notamText(body io.ReadCloser) *notam.Notam {
	notam := notam.NewNotam()

	doc, err := goquery.NewDocumentFromReader(body)
	if err != nil {
		fmt.Println("Unable to retrieve NOTAM info (see NotamText)")
		log.Fatal(err)
	}

	doc.Find(`td[class="txt-notam"]`).Each(
		func(index int, a *goquery.Selection) {
			fillNumber(index, a, notam)
			fillNotamCode(index, a, notam)
			fillIcaoLocation(index, a, notam)
			fillDates(index, a, notam)
			fillText(index, a, notam)
			fillLowerLimit(index, a, notam)
			fillUpperLimit(index, a, notam)
		})
	return notam
}

func fillNotamCode(index int, a *goquery.Selection, notam *notam.Notam) *notam.Notam {

	//Get the NOTAM code identified by Q) and clean it.
	re := regexp.MustCompile("(?s)Q\\).*?\n")
	q := strings.TrimSpace(re.FindString(a.Text()))
	q = strings.TrimRight(q, " \r\n")
	q = strings.TrimLeft(q, "Q)")
	splitted := strings.Split(q, "/")

	notam.NotamCode.Fir = splitted[0]
	notam.NotamCode.Code = splitted[1]
	notam.NotamCode.Traffic = splitted[2]
	notam.NotamCode.Purpose = splitted[3]
	notam.NotamCode.Scope = splitted[4]
	notam.NotamCode.LowerLimit = splitted[5]
	notam.NotamCode.UpperLimit = splitted[6]
	notam.NotamCode.Coordinates = splitted[7]
	fillGeoData(notam)
	return notam
}

func fillGeoData( notam *notam.Notam) *notam.Notam {
	deglat, _ := strconv.Atoi(notam.NotamCode.Coordinates[0:2])
	minlat, _  := strconv.Atoi(notam.NotamCode.Coordinates[2:4])
	hemisphere := notam.NotamCode.Coordinates[4]

	notam.GeoData.Latitude = float64(deglat) + float64(minlat) / 60.0
	if hemisphere == 'S' {
		notam.GeoData.Latitude = -notam.GeoData.Latitude 
	}

	deglong, _ := strconv.Atoi(notam.NotamCode.Coordinates[5:8])
	minlong, _  := strconv.Atoi(notam.NotamCode.Coordinates[8:10])
	side := notam.NotamCode.Coordinates[10]

	notam.GeoData.Longitude = float64(deglong) + float64(minlong) / 60.0
	if side == 'W' {
		notam.GeoData.Longitude = -notam.GeoData.Longitude
	}

	notam.GeoData.Radius,_ = strconv.Atoi(notam.NotamCode.Coordinates[11:14])
	return notam
}

func fillNumber(index int, a *goquery.Selection, notam *notam.Notam) *notam.Notam {
	//Get the NOTAM code identified by Q) and clean it.
	re := regexp.MustCompile("(?s)\\(.*?\n")
	q := strings.TrimSpace(re.FindString(a.Text()))
	q = strings.TrimRight(q, " \r\n")
	q = strings.TrimLeft(q, "(")
	//Usually the NOTAM uses the non break space
	//defines the non break space
	ubkspace := "\xC2\xA0"
	splitted := strings.Split(q, ubkspace)
	if len(splitted) == 1 {
		//if not a success, try with a normal space
		splitted = strings.Split(q, " ")
	}

	if len(splitted) == 1 {
		notam.Number = strings.TrimSpace(splitted[0])
		notam.Status = "Error"
	} else {
		notam.Number = strings.TrimSpace(splitted[0])
		notam.Identifier = strings.TrimSpace(splitted[1])

		if (notam.Identifier == "NOTAMR")  {
			notam.Replace = strings.TrimSpace(splitted[2])
		}
		notam.Status = "Operable"
	}
	return notam
}

func fillIcaoLocation(index int, a *goquery.Selection, notam *notam.Notam) *notam.Notam {
	const ubkspace = "\xC2\xA0"
	//Get the icao location identified by A) and clean it.
	re := regexp.MustCompile("(?s)A\\).*?B\\)")
	q := strings.TrimSpace(re.FindString(a.Text()))
	q = strings.TrimRight(q, "B)")
	q = strings.TrimRight(q, ubkspace)
	q = strings.TrimLeft(q, "A)")
	notam.Icaolocation = q
	return notam
}

func fillDates(index int, a *goquery.Selection, notam *notam.Notam) *notam.Notam {
	const ubkspace = "\xC2\xA0"
	//Get the icao location identified by A) and clean it.
	re := regexp.MustCompile("(?s)B\\).*?C\\).*?(D|E)\\)")
	q := strings.TrimSpace(re.FindString(a.Text()))
	q = strings.TrimLeft(q, "B)")
	q = strings.TrimRight(q, "D)")
	q = strings.TrimRight(q, "\r\n")
	q = strings.TrimRight(q, ubkspace)

	splitted := strings.Split(q, "C)")

	if len(splitted) == 1 {
		notam.Status = "Error"
	} else if len(splitted) == 2 {
		notam.FromDate = splitted[0][0:10]
		notam.ToDate = splitted[1][0:10]
	} else {
		notam.Status = "Error"
	}
	return notam
}

func fillText(index int, a *goquery.Selection, notam *notam.Notam) *notam.Notam {
	const ubkspace = "\xC2\xA0"
	//Get the icao location identified by A) and clean it.
	re := regexp.MustCompile("(?s)E\\).*?(F\\)|G\\)|.*$)")
	q := strings.TrimSpace(re.FindString(a.Text()))
	q = strings.TrimLeft(q, "E)")
	if len(q) < 2 {
		fmt.Printf("Error on the following NOTAM: \n %s \n", a.Text())
	}

	if q[len(q)-2:] == "F)" || q[len(q)-2:] == "G)" {
		q = q[0 : len(q)-2]
	} else {
		q = q[0 : len(q)-1]
	}

	q = strings.TrimRight(q, ubkspace+" \r\n")

	notam.Text = q
	return notam
}

func fillLowerLimit(index int, a *goquery.Selection, notam *notam.Notam) *notam.Notam {
	const ubkspace = "\xC2\xA0"
	//Get the icao location identified by F) and clean it.
	re := regexp.MustCompile("(?s)F\\).*?G\\)")
	q := strings.TrimSpace(re.FindString(a.Text()))
	q = strings.TrimLeft(q, "F)")
	if len(q) > 3 {
		q = q[0 : len(q)-2]
		q = strings.TrimRight(q, ubkspace+" \r\n")
		notam.LowerLimit = q
	} else {
		notam.LowerLimit = ""
	}

	return notam
}

func fillUpperLimit(index int, a *goquery.Selection, notam *notam.Notam) *notam.Notam {
	const ubkspace = "\xC2\xA0"
	//Get the icao location identified by A) and clean it.
	re := regexp.MustCompile("(?s)G\\).*?\\)")
	q := strings.TrimSpace(re.FindString(a.Text()))
	q = strings.TrimLeft(q, "G)")
	if len(q) > 3 {
		q = q[0 : len(q)-1]
		q = strings.TrimRight(q, ubkspace+" \r\n")
		notam.UpperLimit = q
	} else {
		notam.UpperLimit = ""
	}
	return notam
}

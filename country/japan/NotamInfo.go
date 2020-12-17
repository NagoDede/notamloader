package japan

import (
	"errors"
	"fmt"
	"time"
	"github.com/NagoDede/notamloader/notam"
	"io"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
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

// ToUrlValues converts the structure to an url.Values structure.
func (ndf JpNotamDispForm) ToUrlValues() (values url.Values) {
	return structToUrlValues(&ndf)
}

func (ndf JpNotamDispForm) NotamNumber() string {
	nbr, _ := strconv.Atoi(ndf.notam_no)
	year, _ := strconv.Atoi(ndf.notam_year)

	return fmt.Sprintf("%04d", nbr) + "/" + fmt.Sprintf("%02d", year)
}

func (ndf JpNotamDispForm) NotamReference() notam.NotamReference {
	nbr := ndf.NotamNumber()
	return notam.NotamReference{
		Icaolocation: ndf.location,
		Number:       nbr}
}

func (ndf *JpNotamDispForm) RetrieveNotam(httpClient http.Client, url string) *notam.Notam {
	resp, _ := httpClient.PostForm(url, ndf.ToUrlValues())
	notam := notamText(resp.Body)
	resp.Body.Close()
	return notam
}

func postNotamDetail(client http.Client, data url.Values, url string) (resp *http.Response, err error) {

	req, err := http.NewRequest("POST", url, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept-Encoding", "gzip, deflate, br")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9,fr;q=0.8")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Cache-Control", "max-age=0")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.9") //req.AddCookie(client.Jar.Cookies())
	req.Header.Set("Sec-Fetch-Site", "same-origin")
	req.Header.Set("Sec-Fetch-Mode", "navigate")
	req.Header.Set("Sec-Fetch-User", "?1")
	req.Header.Set("Sec-Fetch-Dest", "document")
	req.Header.Set("Upgrade-Insecuree-Requests", "1")
	req.Header.Set("Referer", "https://aisjapan.mlit.go.jp/KeySearcherAction.do")
	return client.Do(req)
}

func notamText(body io.ReadCloser) *notam.Notam {
	notam := notam.NewNotam()

	doc, err := goquery.NewDocumentFromReader(body)
	if err != nil {
		fmt.Println("No url found for navaid extraction")
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

	return notam
}

func fillNumber(index int, a *goquery.Selection, ntm *notam.Notam) *notam.Notam {
	//Get the NOTAM code identified by Q) and clean it.
	re := regexp.MustCompile("(?s)\\(.*?\n")
	str := strings.TrimSpace(re.FindString(a.Text()))
	q := strings.TrimRight(str, " \r\n")
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
		ntm.Number = strings.TrimSpace(splitted[0])
		log.Printf("Unable to retrieve NOTAM identifier from %s for NOTAM %+v", str, ntm.NotamReference)
		ntm.Status = notam.Error
	} else {
		ntm.Number = strings.TrimSpace(splitted[0])
		ntm.Identifier = strings.TrimSpace(splitted[1])

		if (ntm.Identifier == "NOTAMR") || (ntm.Identifier == "NOTAMC") {
			ntm.Replace = strings.TrimSpace(splitted[2])
		}
	}
	return ntm
}

func fillIcaoLocation(index int, a *goquery.Selection, notam *notam.Notam) *notam.Notam {
	//Get the icao location identified by A) and clean it.
	re := regexp.MustCompile("(?s)A\\).*?B\\)")
	q := strings.TrimSpace(re.FindString(a.Text()))
	q = strings.TrimRight(q, "B)")
	q = strings.TrimLeft(q, "A)")
	q = cleanSpaces(q)
	notam.Icaolocation = q
	return notam
}

func dateExtract(s string) (string, error) {
	s = cleanSpaces(s)

	if strings.Contains(s, "PERM") {
		return "PERM", nil
	} else if strings.Contains(s, "UFN") {
		return "UFN", nil
	} else if len(s) >= 10 {
		if strings.Contains(s, "EST") {
			return s[0:13], nil
		} else {
			dateval := s[0:10]
			if _, err := strconv.Atoi(dateval); err == nil {
				return dateval, nil
			} else {
				log.Printf("Date conversion error: retrieved value %s", s)
				return "", errors.New("Date conversion error")
			}
		}
	} else {
		log.Printf("Date conversion error: retrieved value %s", s)
		return "", errors.New("Date conversion error")
	}
}

// Removes the unecessary spaces (including unbreakable spaces)
// of the current string.
// All unbreakable spaces will be replaces by standard space.
func cleanSpaces(s string) string {
	const ubkspace = "\xC2\xA0"
	s = strings.ReplaceAll(s, ubkspace, " ")
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, "  ", " ")
	for strings.Contains(s, "  ") {
		s = strings.ReplaceAll(s, "  ", " ")
	}
	return s
}

func fillDates(index int, a *goquery.Selection, ntm *notam.Notam) *notam.Notam {
	const ubkspace = "\xC2\xA0"
	re := regexp.MustCompile("(?s)B\\).*?C\\).*?(D|E)\\)")
	retrieved := cleanSpaces(re.FindString(a.Text()))
	q := strings.TrimLeft(retrieved, "B)")
	q = strings.TrimRight(q, "E)")
	q = strings.TrimRight(q, "D)")
	q = strings.TrimRight(q, "\r\n")
	q = cleanSpaces(q)

	splitted := strings.Split(q, "C)")
	if len(splitted) == 2 {

		ntmdte, err := dateExtract(splitted[0])
		if err != nil {
			log.Printf("Date conversion error: retrieved fields %s \n notam: %+v \n", retrieved, ntm)
			ntm.Status = notam.Error
		} else {
			ntm.FromDate = ntmdte
		}

		ntmdte, err = dateExtract(splitted[1])
		if err != nil {
			log.Printf("Date conversion error: retrieved fields %s \n notam: %+v \n", retrieved, ntm)
			ntm.Status = notam.Error
		} else {
			ntm.ToDate = ntmdte
		}

		if ntm.FromDate != "PERM" || ntm.FromDate != "UFN" {
			ntm.FromDateDecoded = notam.NotamDateToTime(ntm.FromDate)
		} else {
			//assume permanent validty or UFN  is 10 years
			ntm.FromDateDecoded = time.Now().AddDate(10, 0, 0)
		}
		if ntm.ToDate != "PERM" || ntm.ToDate != "UFN" {
			ntm.ToDateDecoded = notam.NotamDateToTime(ntm.ToDate)
		} else {
			ntm.ToDateDecoded = time.Now().AddDate(10, 0, 0)
		}
	} else {
		log.Printf("Date conversion error: retrieved fields%s \n notam: %+v \n", retrieved, ntm)
		ntm.Status = notam.Error
	}
	return ntm
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

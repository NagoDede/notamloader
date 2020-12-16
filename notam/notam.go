package notam

import (
	"fmt"
	"math"
	"time"
)

const (
	Operable NotamStatus = iota
	Canceled
	Error
)

type Notam struct {
	NotamReference
	Identifier      string `json:"identifier"`
	Replace         string `json:"replace"`
	NotamCode       NotamCode
	FromDate        string
	FromDateDecoded time.Time
	ToDate          string
	ToDateDecoded   time.Time
	Schedule        string
	Text            string
	LowerLimit      string
	UpperLimit      string
	Status          NotamStatus
}

type NotamReference struct {
	Number       string `json:"number"`
	Icaolocation string `json:"icalocation"`
}

type NotamCode struct {
	Fir         string `json:"fir"`
	Code        string `json:"code"`
	Traffic     string `json:"traffic"`
	Purpose     string `json:"purpose"`
	Scope       string `json:"scope"`
	LowerLimit  string `json:"lozerlimit"`
	UpperLimit  string `json:"upperlimit"`
	Coordinates string `json:"coordinates"`
}

type NotamRetriever interface {
	RetrieveNotam() Notam
}

type NotamStatus int

func (s NotamStatus) String() string {
	return [...]string{"Canceled", "Operable"}[s]
}

// Create a new operable NOTAM
func NewNotam() *Notam {
	ntm := new(Notam)
	ntm.Status = Operable
	return ntm
}

func GetNotam(nr NotamRetriever) Notam {
	return nr.RetrieveNotam()
}

// Converts the NOTAM date (yymmddhhmm) to date
// The Golang date parse is limited to
func NotamDateToTime(ndte string) time.Time {
	layout := "0601021504"

	loc, _ := time.LoadLocation("UTC")
	parsedate, _ := time.ParseInLocation(layout, ndte, loc)
	// For layouts specifying the two-digit year 06, a value NN >= 69 will be treated as 19NN and a value NN < 69 will be treated as 20NN.
	if parsedate.Year() < time.Now().Year() {
		var mil float64
		mil = float64(time.Now().Year() / 100.0)
		mil, _ = math.Modf(mil)
		ndte = fmt.Sprintf("%d%s", int(mil), ndte)
		layout := "200601021504"
		parsedate, _ = time.ParseInLocation(layout, ndte, loc)
	}

	return parsedate
}

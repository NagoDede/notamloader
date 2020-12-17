package notam

import (
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
	LowerLimit  string `json:"lowerlimit"`
	UpperLimit  string `json:"upperlimit"`
	Coordinates string `json:"coordinates"`
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

func ValidateNotamDate(s string) bool {
	//(?P<year>\d{2})(?P<month>0[1-9]|1[0-2])(?P<day>0[1-9]|[1-2]\d|3[0-1])(?P<hour>[0-1]\d|2[0-3])(?P<min>[0-5]\d)(\s*EST)*|PERM
	return false
}

package notam
import (
	"sync"
)
type Notam struct {
	Id string `bson:"_id" json:"id,omitempty"`
	NotamReference
	GeoData
	Identifier string `json:"identifier"`
	Replace    string `json:"replace"`
	NotamCode  NotamCode
	FromDate   string
	ToDate     string
	Schedule   string
	Text       string
	LowerLimit string
	UpperLimit string
	Status     string
}

type NotamStatus struct {
	NotamReference
	Status string `json:"status"`
}

type KeyFunc func() string

type NotamReference struct {
	Number       string `json:"number"`
	Icaolocation string `json:"icaolocation"`
	CountryCode  string `json:"countrycode"`
}

func (nr *NotamReference) GetKey() string {
	return nr.CountryCode + "-" + nr.Icaolocation + "-" + nr.Number
}

type GeoData struct {
	Latitude  float64 `json:"Latitude"`
	Longitude float64 `json:"Longitude"`
	Radius    int     `json:"Radius"`
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

type NotamList struct {
	sync.RWMutex
	Data map[string]NotamReference
}

func NewNotam() *Notam {
	return new(Notam)
}

func NewNotamList() *NotamList{
	return &NotamList{Data: make(map[string]NotamReference)}
}

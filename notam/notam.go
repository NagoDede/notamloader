package notam

type Notam struct {
	NotamReference
	Identifier   string `json:"identifier"`
	Replace      string `json:"replace"`
	NotamCode    NotamCode
	FromDate     string
	ToDate       string
	Schedule     string
	Text         string
	LowerLimit   string
	UpperLimit   string
	Status		 string
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

func NewNotam() *Notam {
	return new(Notam)
}
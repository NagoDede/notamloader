// Provides all the structures and fonctions to extract data from a NOTAM
// in text format.
// NOTAM text is usually defined as follow:

// (0221/21 NOTAMR 0220/21
// Q)RJJJ/QGAXX/I/NBO/A/000/999/3515N13655E005
// A)RJNA B)2111031235 C)2111051500
// E)GPS RAIM OUTAGES PREDICTED FOR APCH AS FLW
// 2111041002/2111041010
// 2111050958/2111051006)

// or

// LFFA-M3750/21
// Q) LFXX/QSULT/ I/NBO/ E/295/999/4815N00044E065
// A) LFFF LFRR
// B) 2021 Nov 03  23:00 C) 2021 Nov 04  03:30
// E) FREQUENCE UHF 389.875MHZ INDISPONIBLE. PAS DE TRAFIC NON EQUIPE
// 8.33MHZ ACCEPTE DANS LES SECTEURS XU, XI ET XS DE BREST.

// The fields Q), A) B) C) D) E) F) and G) are defined in the ICAO manual.
// The Notam identification is specific to the country. For our case, this is acheived by
// defining a function FillNotamNumber that will be used to fill the NotamReference field.

package notam

import (
	"fmt"
	_"log"
	_ "net/http"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/NagoDede/notamloader/webclient"
)

type Notam struct {
	Id string `bson:"_id" json:"id,omitempty"`
	NotamReference
	GeoData
	Identifier string `json:"identifier"`
	Replace    string `json:"replace"`
	NotamCode  NotamCode
	FromDate   string
	FromDateUtcTime	time.Time
	FromDateUtcClear	string
	ToDate     string
	ToDateUtcTime	time.Time 
	ToDateUtcClear	string
	Schedule   string
	Text       string
	LowerLimit string
	UpperLimit string
	Status     string
}

type NotamAdvanced struct {
	Notam

	FillIcaoLocation func(*NotamAdvanced, string) *NotamAdvanced `json:"-"`
	FillNotamCode    func(*NotamAdvanced, string) *NotamAdvanced `json:"-"`
	FillNotamNumber  func(*NotamAdvanced, string) *NotamAdvanced `json:"-"`
	FillDates        func(*NotamAdvanced, string) *NotamAdvanced `json:"-"`
	FillText         func(*NotamAdvanced, string) *NotamAdvanced `json:"-"`
	FillLowerLimit   func(*NotamAdvanced, string) *NotamAdvanced `json:"-"`
	FillUpperLimit   func(*NotamAdvanced, string) *NotamAdvanced `json:"-"`
	FillKey			func(*NotamAdvanced) *NotamAdvanced `json:"-"`
}

type NotamStatus struct {
	NotamReference
	Status string `json:"status"`
}

type INotam interface {
	FillNotamNumber(string)
	FillNotamCode(string)
	FillIcaoLocation(string)
	FillDates(string)
	FillText(string)
	FillLowerLimit(string)
	FillUpperLimit(string)
	FillKey()
}

type KeyFunc func() string

type NotamReference struct {
	Number       string `json:"number"`
	Icaolocation string `json:"icaolocation"`
	AfsCode      string `json:"afscode"`
	FirCode      string `json:"fircode"`
}

func (nr *NotamReference) GetKey() string {
	//return nr.CountryCode + "-" + nr.Icaolocation + "-" + nr.Number
	// if nr.FirCode != "" {
	// 	return nr.AfsCode + "-" + nr.FirCode + "-" + nr.Number
	// }

	// if nr.Icaolocation != "" {
	// 	return nr.AfsCode + "-" + nr.Icaolocation + "-" + nr.Number
	// }

	return nr.AfsCode + "-" + nr.Number
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

const ubkspace = "\xC2\xA0"

func NewNotam() *Notam {
	return new(Notam)
}

func NewNotamAdvanced() *NotamAdvanced {
	ntm := new(NotamAdvanced)
	ntm.FillDates = FillDates
	ntm.FillIcaoLocation = FillIcaoLocation
	ntm.FillLowerLimit = FillLowerLimit
	ntm.FillNotamCode = FillNotamCode
	ntm.FillNotamNumber = FillNotamNumber
	ntm.FillText = FillText
	ntm.FillUpperLimit = FillUpperLimit
	ntm.FillKey = FillKey
	return ntm
}

 func NewNotamList() *NotamList {
 	return &NotamList{Data: make(map[string]NotamReference)}
 }

func FillNotamFromText(ntm *NotamAdvanced, notamText string) *NotamAdvanced {

	ntm = ntm.FillNotamNumber(ntm, notamText)
	ntm = ntm.FillNotamCode(ntm, notamText)
	ntm = ntm.FillIcaoLocation(ntm, notamText)
	ntm = ntm.FillDates(ntm, notamText)
	ntm = ntm.FillText(ntm, notamText)
	ntm = ntm.FillLowerLimit(ntm, notamText)
	ntm = ntm.FillUpperLimit(ntm, notamText)
	ntm = ntm.FillKey(ntm)
	ntm.fillFromDates()
	ntm.fillToDates()
	return ntm
}

func (ntm *NotamAdvanced) fillFromDates(){
	if len(ntm.FromDate) == 10 {
		parsed, err :=  NotamDateToTime(ntm.FromDate, time.UTC)//time.Parse("0601021504", sDateFrom)
		if err !=nil {
			fmt.Printf("Err to convert date %s \n", ntm.FromDate)
		}
		ntm.FromDateUtcTime = parsed
		ntm.FromDateUtcClear = parsed.Format("Mon Jan 2 15:04 UTC 2006")

	} else if len(ntm.FromDate) == 13 {
		tz := ntm.FromDate[len(ntm.FromDate)-3:]
		location, err := time.LoadLocation(tz)
		if err != nil {
			fmt.Printf("Not a timezone %s \n", tz)
		}
		parsed, err := NotamDateToTime(ntm.FromDate[0:10], location)//time.ParseInLocation("0601021504", sDateFrom[0:10], location)
		if err !=nil {
			fmt.Printf("Err to convert date %s (loca %s) \n", ntm.FromDate[0:10], ntm.FromDate[10:])
		}
		ntm.FromDateUtcTime = parsed.UTC()
		ntm.FromDateUtcClear = ntm.FromDateUtcTime.Format("Mon Jan 2 15:04 UTC 2006")
	} else {
		fmt.Printf("%s not a valid date To \n", ntm.FromDate)
	}
}

func (ntm *NotamAdvanced) fillToDates(){
	if ntm.ToDate == "PERM" {
		ntm.ToDateUtcClear = "Permanent"
	} else if len(ntm.ToDate) == 10 {
		parsed, err :=   NotamDateToTime(ntm.ToDate, time.UTC)//time.Parse("0601021504", sDateFrom)
		if err !=nil {
			fmt.Printf("Err to convert date %s \n", ntm.FromDate)
		}
		ntm.ToDateUtcTime = parsed
		ntm.ToDateUtcClear = parsed.Format("Mon Jan 2 15:04 UTC 2006")

	} else if len(ntm.ToDate) == 13 {
		tz := ntm.ToDate[len(ntm.ToDate)-3:]
		location, err := time.LoadLocation(tz)
		if err != nil {
			fmt.Printf("Not a timezone %s \n", tz)
		}
		parsed, err := NotamDateToTime(ntm.ToDate[0:10], location)//time.ParseInLocation("0601021504", sDateFrom[0:10], location)
		if err !=nil {
			fmt.Printf("Err to convert date %s (loca %s) \n", ntm.ToDate[0:10], ntm.ToDate[10:])		}
		ntm.ToDateUtcTime = parsed.UTC()
		ntm.ToDateUtcClear = ntm.ToDateUtcTime.Format("Mon Jan 2 15:04 UTC 2006")
	} else {
		fmt.Printf("%s not a valid date To \n", ntm.ToDate)
	}
}

func FillKey(ntm *NotamAdvanced) *NotamAdvanced {
	ntm.Id = ntm.GetKey()
	return ntm
}

func FillNotamNumber(ntm *NotamAdvanced, txt string) *NotamAdvanced {
	txt = txt[:strings.Index(txt, "Q)")+6] //keep text up to the QCode to get the Fir
	txt = strings.Trim(txt, " \r\n\t")

	//For france, the airport code is not used
	//fr.NotamReference.Icaolocation = txt[:strings.Index(txt, "-")]
	end := strings.Index(txt, " ")
	if strings.Index(txt, "\n") < end {
		end = strings.Index(txt, "\n")
	}
	if end <= 0 {
		end = len(txt)
	}
	ntm.NotamReference.Number = strings.Trim(txt[strings.Index(txt, "-")+1:end], " \r\n\t")

	return ntm
}

///Retrieve and extract the codes defined in the Q) parameters
//Codes are defined by Q)LFFF/QWPLW/IV/M/AW/000/125/4932N00005E005
//where LFFF is the FIR
//	QWPLW is QCode, see ICAO DOC 8216.
//		Q is an ID.
//		2nd and 3rd letters are related to the NOTAM subject
// 		4th and 5th letters state or conditions related to the subject.
// IV is the traffic
// M is the NOTAM object (N, B, O, M)
// AW is related to the range
//		A Airport
//		E En Route
//		W Navigation warning
//		AE Airport / En route
//		AW Airport and Navigation warning
//	000 Altitude INf
//	999 Altitude Sup
// 	4932N00005E005 Coordinates and influence radius
func FillNotamCode(ntm *NotamAdvanced, txt string) *NotamAdvanced {
	re := regexp.MustCompile("(?s)Q\\).*?(\n|A\\))") //the Q) parameters is defined on a single line or up to the next field A)
	q := strings.TrimSpace(re.FindString(txt))
	q = strings.ReplaceAll(q, "Q)", "")
	q = strings.ReplaceAll(q, "A)", "")
	//clean possible spaces
	for strings.Index(q, " ") > 0 {
		q = strings.ReplaceAll(q, " ", "")
	}
	q = strings.TrimRight(q, " \r\n") //remove all the unecessary items on the right
	splitted := strings.Split(q, "/") //the code separation is a /

	ntm.NotamCode.Fir = strings.Trim(splitted[0], " ")
	ntm.NotamReference.FirCode = ntm.NotamCode.Fir
	ntm.NotamCode.Code = strings.Trim(splitted[1], " ")
	ntm.NotamCode.Traffic = strings.Trim(splitted[2], " ")
	ntm.NotamCode.Purpose = strings.Trim(splitted[3], " ")
	ntm.NotamCode.Scope = strings.Trim(splitted[4], " ")
	ntm.NotamCode.LowerLimit = strings.Trim(splitted[5], " ")
	ntm.NotamCode.UpperLimit = strings.Trim(splitted[6], " ")
	ntm.NotamCode.Coordinates = strings.Trim(splitted[7], " ")

	ntm.FillGeoData()
	return ntm
}

func (notam *Notam) FillGeoData() *Notam {
	//if there is no coordinate, unable to compute the geodata, so return the same object
	if notam.NotamCode.Coordinates == "" {
		return notam
	}
	deglat, err := strconv.Atoi(notam.NotamCode.Coordinates[0:2])
	if err != nil {
		deglat = 0.0
	}
	minlat, err := strconv.Atoi(notam.NotamCode.Coordinates[2:4])
	if err != nil {
		minlat = 0.0
	}
	hemisphere := notam.NotamCode.Coordinates[4]

	notam.GeoData.Latitude = float64(deglat) + float64(minlat)/60.0
	if hemisphere == 'S' {
		notam.GeoData.Latitude = -notam.GeoData.Latitude
	}

	deglong, err := strconv.Atoi(notam.NotamCode.Coordinates[5:8])
	if err != nil {
		deglong = 0.0
	}
	minlong, err := strconv.Atoi(notam.NotamCode.Coordinates[8:10])
	if err != nil {
		minlong = 0.0
	}
	side := notam.NotamCode.Coordinates[10]

	notam.GeoData.Longitude = float64(deglong) + float64(minlong)/60.0
	if side == 'W' {
		notam.GeoData.Longitude = -notam.GeoData.Longitude
	}

	if len(notam.NotamCode.Coordinates) > 11 {
		notam.GeoData.Radius, err = strconv.Atoi(notam.NotamCode.Coordinates[11:14])
		if err != nil {
			notam.GeoData.Radius = 1
		}
	}
	return notam
}

func FillIcaoLocation(ntm *NotamAdvanced, txt string) *NotamAdvanced {

	//Get the icao location identified by A) and clean it.
	re := regexp.MustCompile("(?s)A\\).*?B\\)")
	q := strings.TrimSpace(re.FindString(txt))
	q = strings.TrimRight(q, "B)")
	q = strings.TrimRight(q, ubkspace)
	q = strings.TrimLeft(q, "A)")
	q = strings.Trim(q, " \r\n\t")
	q = strings.ReplaceAll(q, ubkspace, " ")
	q = strings.ReplaceAll(q, "  ", " ")
	ntm.Icaolocation = q

	//if the location was not set previously, now we have at least an airpot info, so, we can perform a request
	if ntm.NotamCode.Coordinates == "" {
		fmt.Println("!! Retrieve Airport Position data through airport-data.com for " + ntm.Icaolocation)
		aptData := webclient.GetAirportData(ntm.Icaolocation)
		if aptData.Status == 200 {
			latitude, err := strconv.ParseFloat(aptData.Latitude, 64)
			if err != nil {
				fmt.Printf("Unable to retrieve and convert Latitude %s for %s \n", aptData.Latitude, ntm.Icaolocation)
			} else 			{
				ntm.GeoData.Latitude = latitude
			}

			longitude, err := strconv.ParseFloat(aptData.Longitude, 64)
			if err != nil {
				fmt.Printf("Unable to retrieve and convert Longitude %s for %s \n", aptData.Longitude, ntm.Icaolocation)
			} else 			{
				ntm.GeoData.Longitude = longitude
			}
		}
	}

	return ntm
}

func FillDates(fr *NotamAdvanced, txt string) *NotamAdvanced {

	cutAtSpaceOrCr := func(s string) string {
		space := strings.Index(s, " ")
		cr := strings.Index(s, "\n")
		if (space > 0) && (cr > space) {
			s = s[:space]
		} else {
			if cr > 0 {
				s = s[:cr]
			} else {
				return s
			}
		}
		return s
	}

	//retrieve the Date fields and clean around
	const ubkspace = "\xC2\xA0"
	re := regexp.MustCompile("(?s)B\\).*?C\\).*?(D|E)\\)")
	q := strings.TrimSpace(re.FindString(txt))
	q = strings.TrimLeft(q, "B)")
	q = strings.TrimRight(q, "D)")
	q = strings.TrimRight(q, " \r\n\t")
	q = strings.TrimRight(q, ubkspace)
	q = strings.ReplaceAll(q, ubkspace, " ")
	for strings.Contains(q, "  ") {
		q = strings.ReplaceAll(q, "  ", " ")
	}
	splitted := strings.Split(q, "C)")

	if len(splitted) == 1 {
		fr.Status = "Error"
	} else if len(splitted) == 2 {
		sDateFrom := splitted[0]
		sDateFrom = strings.Trim(sDateFrom, " \n\r\t")
		sDateFrom = cutAtSpaceOrCr(sDateFrom)

		fr.FromDate = sDateFrom
	}


		sDateTo := splitted[1]
		sDateTo = strings.Trim(sDateTo, " \n\r\t")
		sDateTo = cutAtSpaceOrCr(sDateTo)
		//NOTAM for AIP references are indicated as PERManent.
		if strings.Contains(sDateTo, "PERM") {
			fr.ToDate = "PERM"
		} else {
			fr.ToDate = sDateTo	
		}		

	return fr
}


func FillText(ntm *NotamAdvanced, txt string) *NotamAdvanced {
	const ubkspace = "\xC2\xA0"
	//Get the icao location identified by A) and clean it.
	re := regexp.MustCompile("(?s)E\\).*?(F\\)|G\\)|.*$)")
	q := strings.TrimSpace(re.FindString(txt))
	q = strings.TrimLeft(q, "E)")
	if len(q) < 2 {
		fmt.Printf("Error on the following NOTAM: \n %s \n", txt)
	}

	if q[len(q)-2:] == "F)" || q[len(q)-2:] == "G)" {
		q = q[0 : len(q)-2]
	} 

	q = strings.Trim(q, ubkspace + " \r\n\t")

	ntm.Text = q
	return ntm
}

func FillLowerLimit(ntm *NotamAdvanced, txt string) *NotamAdvanced {
	const ubkspace = "\xC2\xA0"
	//Get the icao location identified by F) and clean it.
	re := regexp.MustCompile("(?s)F\\).*?G\\)")
	q := strings.TrimSpace(re.FindString(txt))
	q = strings.TrimLeft(q, "F)")
	if len(q) > 3 {
		q = q[0 : len(q)-2] //remove the G)
		q = strings.Trim(q, ubkspace+" \r\n")
		ntm.LowerLimit = q
	} else {
		ntm.LowerLimit = ""
	}
	return ntm
}

func FillUpperLimit(ntm *NotamAdvanced, txt string) *NotamAdvanced {
	const ubkspace = "\xC2\xA0"
	//Get the icao location identified by A) and clean it.
	re := regexp.MustCompile("(?s)G\\).*?(\\)|\\z)")
	q := strings.TrimSpace(re.FindString(txt))
	q = strings.TrimLeft(q, "G)")
	if len(q) > 3 {
		q = strings.Trim(q, ubkspace+" \r\n\t")
		ntm.UpperLimit = q
	} else {
		ntm.UpperLimit = ""
	}
	return ntm
}

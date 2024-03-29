package france

import (
	_ "errors"
	_ "fmt"
	_ "io"
	_ "io/ioutil"
	_ "log"
	_ "net/http"
	_ "net/http"
	"net/url"
	"strconv"
	_ "strings"

	"github.com/NagoDede/notamloader/webclient"
	_ "github.com/NagoDede/notamloader/webclient"
	_ "github.com/PuerkitoBio/goquery"
)

type FirFormRequest struct{
	Resultat 		bool 	`json:"bResultat"`
	Impression 		string	`json:"bImpression"`
	ModeAffichage 	string // COMPLET
	FIR_Date_DATE 	string //2021/10/28
	FIR_Date_HEURE 	string //: 19:54
	FIR_Langue		string //: EN
	FIR_Duree		string //: 12
	FIR_CM_REGLE	string //: 1
	FIR_CM_GPS 		string	//: 1
	FIR_CM_INFO_COMP string //: 1
	FIR_CM_ROUTE 	string//: 1
	FIR_NivMin		string//: 0
	FIR_NivMax		string//: 999
	FIR_Tab_Fir[10]	string//: LFRR
}

type AptFormRequest struct{
	Resultat 		bool 	`json:"bResultat"`
	Impression 		string	`json:"bImpression"`
	ModeAffichage 	string // COMPLET
	AERO_Date_DATE 	string //2021/10/28
	AERO_Date_HEURE 	string //: 19:54
	AERO_Langue		string //: EN
	AERO_Duree		string //: 12
	AERO_CM_REGLE	string //: 1
	AERO_CM_GPS 		string	//: 1
	AERO_CM_INFO_COMP string //: 1
	AERO_Rayon	string
	AERO_Plafond	string
	AERO_Tab_Fir[12]	string//: LFRR
}

func NewAptFormResumeRequest(apts[12] string, sDate string, sHour string) *AptFormRequest{
	var tmp =  &AptFormRequest{
		Resultat : true,
		Impression 	: "",
		ModeAffichage: 	"COMPLET",
		AERO_Date_DATE: 	sDate,
		AERO_Date_HEURE: 	sHour,
		AERO_Langue:		"EN",
		AERO_Duree:		"48",
		AERO_CM_REGLE:	"1",
		AERO_CM_GPS: 		"1",
		AERO_CM_INFO_COMP: "1",
		AERO_Rayon:	"30",
		AERO_Plafond:	"115",
	}

	for i,a := range apts{
		tmp.AERO_Tab_Fir[i] = a
	}

	return tmp
}

func NewFirFormResumeRequest(icaoCode string, sDate string, sHour string) *FirFormRequest{
	return &FirFormRequest{
		Resultat: true,
		Impression: "",
		ModeAffichage: "RESUME",
		FIR_Date_DATE: sDate,
		FIR_Date_HEURE: sHour,
		FIR_Langue: "EN",
		FIR_Duree: "48",
		FIR_CM_REGLE: "1",
		FIR_CM_GPS: "1",
		FIR_CM_INFO_COMP: "1",
		FIR_CM_ROUTE: "1",
		FIR_NivMin: "0",
		FIR_NivMax: "999",
		FIR_Tab_Fir: [10]string{icaoCode},
	}
}

func (form *FirFormRequest) Encode() (url.Values) {
	values := webclient.StructToMap(form)
	values.Add("bImpression","")
	values.Add("bResultat","true")
	values.Del("FIR_Tab_Fir")
	values.Del("Impression")
	values.Del("Resultat")
	for i:=0; i<10; i++ {
		values.Add("FIR_Tab_Fir[" + strconv.Itoa(i) + "]", form.FIR_Tab_Fir[i])
	}
	return values
}

func (form *FirFormRequest) EncodeForComplet(notamMin int, count int) (url.Values) {
	values := webclient.StructToMap(form)
	values.Add("bImpression","")
	values.Add("bResultat","true")
	values.Add("bResaisir","false")
	values.Del("FIR_Tab_Fir")
	values.Del("Impression")
	values.Del("Resultat")
	for i:=0; i<10; i++ {
		values.Add("FIR_Tab_Fir[" + strconv.Itoa(i) + "]", form.FIR_Tab_Fir[i])
	}

	if count == 0 {
		values.Add("NOTAM[" + strconv.Itoa(notamMin) + "]", "on")
	} else {
	for i:=notamMin; i<(notamMin+count); i++ {
		values.Add("NOTAM[" + strconv.Itoa(i) + "]", "on")
	} 
	}
	return values
}
package asecna

import (
	"net/url"
	_ "strings"

	"github.com/NagoDede/notamloader/webclient"

)

type FormRequest struct{
	Bni 		string 	`json:"qr_bni"`
	Fir 		string	`json:"qr_qfir"`
	Firx 		string 	`json:"qr_qfirx"`
	Number	 	string 	`json:"qr_num"`
	DateArrStart 	string `json:"qr_datearrd"`
	DateArrEnd		string `json:"qr_datearrf"`
	DateValidStart		string `json:"qr_datevald"`
	DateValidEnd		string `json:"qr_datevalf"`
	Text 		string	`json:"qr_texte"`
	Maxrows		int `json:"qr_maxrows"`
	Submit 		string	`json:"submit"`
}

func NewFormRequest(firCode string) *FormRequest{
	return &FormRequest{
		Bni: "TOUT",
		Fir: firCode,
		Maxrows: 100,
		Submit: "Consulter",
	}
}

func (form *FormRequest) Encode() (url.Values) {
	values := webclient.StructToMap(form)
	return values
}

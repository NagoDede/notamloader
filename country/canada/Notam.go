package canada

import (
	_"fmt"
	_ "log"
	_ "regexp"
	"strings"
	_ "time"

	_"github.com/NagoDede/notamloader/database"
	_ "github.com/NagoDede/notamloader/database"
	"github.com/NagoDede/notamloader/notam"
)

type Notam struct {
	*notam.NotamAdvanced
}

func NewNotam(afs string) *Notam {
	frntm := &Notam{NotamAdvanced: notam.NewNotamAdvanced()}
	frntm.NotamAdvanced.FillNotamNumber = FillNotamNumber
	//frntm.NotamAdvanced.FillDates = FillDates
	frntm.NotamReference.AfsCode = afs
	return frntm
}

func FillNotamNumber(fr *notam.NotamAdvanced, txt string) *notam.NotamAdvanced {

	fir := txt[strings.Index(txt, "Q)")+2 : strings.Index(txt, "Q)")+6]
	txt = txt[:strings.Index(txt, "Q)")+6] //keep text up to the QCode
	txt = strings.Trim(txt, " \r\n\t")
	txt = txt[strings.Index(txt, "(")+1:]
	fr.NotamReference.Icaolocation = fir
	end := strings.Index(txt, " ")
	fr.NotamReference.Number = strings.Trim(txt[:end], " \r\n\t")

	return fr
}



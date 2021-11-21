package asecna

import (

	"regexp"
	"strings"
	"time"

	_"github.com/NagoDede/notamloader/database"
	"github.com/NagoDede/notamloader/notam"
)

type Notam struct {
	*notam.NotamAdvanced
}


func NewNotam(afs string) *Notam {
	frntm := &Notam{NotamAdvanced: notam.NewNotamAdvanced()}
	frntm.NotamAdvanced.FillNotamNumber = FillNotamNumber
	frntm.NotamReference.AfsCode = afs
	
	return frntm
}

func FillNotamNumber(fr *notam.NotamAdvanced, txt string) *notam.NotamAdvanced {
	fir := txt[strings.Index(txt, "Q)")+2:strings.Index(txt, "Q)")+6]
	txt = txt[:strings.Index(txt, "Q)")+6] //keep text up to the QCode
	txt = strings.Trim(txt, " \r\n\t")
	txt = txt[strings.Index(txt, "(")+1:]
	fr.NotamReference.Icaolocation = fir
		end := strings.Index(txt, " ")
	fr.NotamReference.Number = strings.Trim(txt[:end], " \r\n\t")
	Logger.Trace().Msgf("Create Notam: %s ", fr.NotamReference.Number)
	return fr
}



func FillDates(fr *notam.NotamAdvanced, txt string) *notam.NotamAdvanced {
	const ubkspace = "\xC2\xA0"
	re := regexp.MustCompile("(?s)B\\).*?C\\).*?(D|E)\\)")
	q := strings.TrimSpace(re.FindString(txt))
	q = strings.ReplaceAll(q, "B)", "")
	q = strings.ReplaceAll(q, "E)","")
	q = strings.ReplaceAll(q, ubkspace, " ")
	q = strings.Trim(q, " \r\n\t")

	for strings.Contains(q,"  ") {
		q = strings.ReplaceAll(q, "  ", " ")
	}

	splitted := strings.Split(q, "C)")

	if len(splitted) == 1 {
		fr.Status = "Error"
	} else if len(splitted) == 2 {
		sDateFrom := splitted[0]
		sDateFrom = strings.ReplaceAll(sDateFrom, "  ", " ")
		sDateFrom = strings.Trim(sDateFrom, " \n\r\t")
		//2021 Jan 27 23:59
		//--> 2006 Jan 02 15:04
		dateFrom, _ := time.Parse("2006-01-02 15:04:05", sDateFrom)
		fr.FromDate = dateFrom.Format("0601021504")
	
		sDateTo := splitted[1]
		sDateTo = strings.ReplaceAll(sDateTo, "  ", " ")
		//NOTAM for AIP references are indicated as PERManent.
		if strings.Contains(sDateTo, "PERM") {
			fr.ToDate = "PERM"
		} else {
			sDateTo = strings.Trim(sDateTo[0:18], " \n\r\t")
			dateTo, _ := time.Parse("2006-01-02 15:04:05", sDateTo)
			fr.ToDate = dateTo.Format("0601021504")

		}
	} else {
		fr.Status = "Error"
	}

	Logger.Trace().Msgf("Notam %s From Date: %s to date: %s", fr.Number, fr.FromDate, fr.ToDate)
	return fr
}



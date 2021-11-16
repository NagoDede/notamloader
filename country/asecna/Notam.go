package asecna

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/NagoDede/notamloader/database"
	"github.com/NagoDede/notamloader/notam"
)

type Notam struct {
	*notam.NotamAdvanced
}

 type NotamList struct {
 	//notamList []*FranceNotam
 	notamList map[string]*Notam
 }

 func NewNotamList() *NotamList {
 	list := &(NotamList{})
 	list.notamList = make(map[string]*Notam) // []*FranceNotam{}
 	return list
 }

func NewNotam(afs string) *Notam {
	frntm := &Notam{NotamAdvanced: notam.NewNotamAdvanced()}
	frntm.NotamAdvanced.FillNotamNumber = FillNotamNumber
	frntm.NotamAdvanced.FillDates = FillDates
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
	
	return fr
}

func FillDates(fr *notam.NotamAdvanced, txt string) *notam.NotamAdvanced {
	const ubkspace = "\xC2\xA0"
	re := regexp.MustCompile("(?s)B\\).*?C\\).*?(D|E)\\)")
	q := strings.TrimSpace(re.FindString(txt))
	q = strings.TrimLeft(q, "B)")
	q = strings.TrimRight(q, "D)")
	q = strings.TrimRight(q, " \r\n\t")
	q = strings.TrimRight(q, ubkspace)
	q = strings.ReplaceAll(q, ubkspace, " ")
	splitted := strings.Split(q, "C)")

	if len(splitted) == 1 {
		fr.Status = "Error"
	} else if len(splitted) == 2 {
		sDateFrom := splitted[0]
		sDateFrom = strings.ReplaceAll(sDateFrom, "  ", " ")
		sDateFrom = strings.Trim(sDateFrom, " \n\r\t")
		//2021 Jan 27 23:59
		//--> 2006 Jan 02 15:04
		dateFrom, _ := time.Parse("2006 Jan 02 15:04", sDateFrom)
		fr.FromDate = dateFrom.Format("0601021504")
		sDateTo := splitted[1]
		sDateTo = strings.ReplaceAll(sDateTo, "  ", " ")
		//NOTAM for AIP references are indicated as PERManent.
		if strings.Contains(sDateTo, "PERM") {
			fr.ToDate = "PERM"
		} else {
			sDateTo = strings.Trim(sDateTo[0:18], " \n\r\t")
			dateTo, _ := time.Parse("2006 Jan 02 15:04", sDateTo)
			fr.ToDate = dateTo.Format("0601021504")
		}
	} else {
		fr.Status = "Error"
	}
	return fr
}

func (fl *NotamList) SendToDatabase(mg *database.Mongodb) *notam.NotamReferenceList {

	notamList := notam.NewNotamReferenceList()
	for _, frNotam := range fl.notamList {
		frNotam.Status = "Operable"

		//avoid duplicate
		_, ok := notamList.Data[frNotam.Id]
		if !ok {
			//record all notams, except the duplicate
			notamList.Data[frNotam.Id] = frNotam.NotamReference

			//send to db only if necessary
			_, isOld := mg.ActiveNotams[frNotam.Id]
			if !isOld {
				//fmt.Printf("Write %s / %d \n", i, len(fl.notamList))
				mg.AddNotam(&frNotam.Notam)
			}
		} else {
			fmt.Printf("Duplicated %s \n", frNotam.Id)
		}
	}
	return notamList
}

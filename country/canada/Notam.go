package canada

import (
	"fmt"
	_ "log"
	_ "regexp"
	"strings"
	_ "time"

	"github.com/NagoDede/notamloader/database"
	_ "github.com/NagoDede/notamloader/database"
	"github.com/NagoDede/notamloader/notam"
)

type Notam struct {
	*notam.NotamAdvanced
}

 type NotamList struct {
 	Data map[string]*Notam
 }

 func NewNotamList() *NotamList {
 	list := &(NotamList{})
 	list.Data = make(map[string]*Notam) // []*FranceNotam{}
 	return list
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

 func (fl *NotamList) SendToDatabase(mg *database.Mongodb) *notam.NotamList {

 	notamList := notam.NewNotamList()
 	for _, frNotam := range fl.Data {
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

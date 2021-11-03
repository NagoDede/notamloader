package france

import (
	_ "context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	_ "log"
	"regexp"
	"strings"
	"time"

	_ "net"
	_ "net/http"
	_ "net/http/cookiejar"
	_ "net/url"
	"os"
	_ "reflect"
	_ "strings"
	_ "sync"

	"github.com/NagoDede/notamloader/database"
	"github.com/NagoDede/notamloader/notam"
	"github.com/NagoDede/notamloader/webclient"
	_ "github.com/NagoDede/notamloader/webclient"
	_ "go.mongodb.org/mongo-driver/mongo"
	_ "golang.org/x/net/publicsuffix"
)

var FranceAis DefData

// FrData contains all the information required to connect and retrieve NOTAM from AIS services
type DefData struct {
	NotamRequestUrl	string	`json:"notamRequestUrl`
	CountryCode      string          `json:"countryCode"`
	CodeListPath     string          `json:"codeListPath"`
	RequiredLocation []string        `json:"requiredLocation"`
}

type FranceNotam struct{
	*notam.NotamAdvanced
}

func NewFranceNotam() *FranceNotam{
	frntm :=  &FranceNotam{ NotamAdvanced: notam.NewNotamAdvanced()}
	frntm.NotamAdvanced.FillNotamNumber = FillNotamNumber
	frntm.NotamAdvanced.FillDates = FillDates
	return frntm
}

func FillNotamNumber(fr *notam.NotamAdvanced, txt string)  *notam.NotamAdvanced {
	fr.NotamReference.CountryCode = "FRA"
	//fr.NotamReference.Icaolocation = fr.NotamCode.Fir
	fr.NotamReference.Number = txt[5:13]
	return fr
}

func FillDates(fr *notam.NotamAdvanced, txt string)  *notam.NotamAdvanced {
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
		sDateFrom = strings.ReplaceAll(sDateFrom,"  ", " ")
		sDateFrom = strings.Trim(sDateFrom, " \n\r\t")
		//2021 Jan 27 23:59 
		//--> 2006 Jan 02 15:04
		dateFrom, _ := time.Parse("2006 Jan 02 15:04", sDateFrom)
		fr.FromDate = dateFrom.Format("0601021504")
		sDateTo := splitted[1]
		sDateTo = strings.ReplaceAll(sDateTo,"  ", " ")
		sDateTo = strings.Trim(sDateTo[0:18], " \n\r\t")
		dateTo, _ := time.Parse("2006 Jan 02 15:04", sDateTo)
		fr.ToDate = dateTo.Format("0601021504")
	} else {
		fr.Status = "Error"
	}
	return fr

}

var mongoClient *database.Mongodb
var aisClient *webclient.AisWebClient

// Process launches the global process to recover the NOTAMs from the Japan AIS webpages.
// It recovers the relevant information from a json file, set in ./country/japan/def.json.
// Then, it initiates the http and mongodb interfaces.
// Once achieved, it interrogates the web form by providing the location ICAO code to
// the webform to identify the reference list of the relevant NOTAM.
func (def *DefData) Process() {

	//retrieve the configuration data from the json file
	def.loadJsonFile("./country/france/def.json")
	//Init a the http client thanks tp the configuration data
//	mainHttpClient := webclient.NewAisWebClient()//newHttpClient()
	//enrouteHttpClient := newAisWebClient()//newHttpClient()
	fmt.Println("Connected to AIS France")

	//Will contain all the retrieved Notams
	//notams := notam.NewNotamList()
	ntm := NewFranceNotam()
	//ntm.FillNotamFromText("dede")

	//Initiate a new mongo db interface
	mongoClient = database.NewMongoDb(def.CountryCode)
	aisClient = webclient.NewAisWebClient()

	//def.RetrieveAllNotams()
	txt := "LFFA-F0092/21 \n Q) LFXX/QAFXX/IV/NBO/ E/000/999/4412N00040E460\nA) LFBB LFEE LFFF LFMM LFRR \n	B) 2021 Jan 27  23:59 C) 2021 Dec 31  23:59\nE) AFIN DE GARANTIR UNE LIVRAISON SURE ET RAPIDE DES VACCINS COVID-19,	LES EXPLOITANTS D AERONEFS TRANSPORTANT CES VACCINS DOIVENT DEMANDER	L APPROBATION DE L EXEMPTION DES MESURES ATFM A LA DGAC POUR CHAQUE	 VOL JUGE CRITIQUE SELON LA PROCEDURE DECRITE DANS L AIP FRANCE	 1.9.3.3.1. APRES APPROBATION, STS/ATFMX ET RMK/VACCINE DOIVENT ETRE	 INSERES DANS LA CASE 18 DU PLAN DE VOL. 	LES EXPLOITANTS D AERONEFS TRANSPORTANT REGULIEREMENT DES VACCINS	COVID-19 PEUVENT DEMANDER L APPROBATION A L AVANCE POUR TOUS LES VOLS	CONCERNES."
	ntm.NotamAdvanced = notam.FillNotamFromText(ntm.NotamAdvanced, txt)
	fmt.Printf("notam: %+v \n", ntm.Notam)


//	fmt.Printf("Current NOTAMs: %d \n", len(notams.Data))
	// //Once all the NOTAM havebeen identified, identify the deleted ones and set them in the db.
	// canceledNotams := mongoClient.IdentifyCanceledNotams(notams.Data)
	// fmt.Printf("Canceled NOTAM: %d \n", len(*canceledNotams))
	// mongoClient.SetCanceledNotamList(canceledNotams)
	// mongoClient.WriteActiveNotamToFile("./web/notams/japan.json")
}

/*
Load the JSON file used for the definition.
*/
func (def *DefData) loadJsonFile(path string) {
	// Open our jsonFile
	jsonFile, err := os.Open(path)
	// if we os.Open returns an error then handle it
	if err != nil {
		fmt.Println(err)
	}

	// defer the closing of our jsonFile so that we can parse it later on
	defer jsonFile.Close()

	byteValue, err := ioutil.ReadAll(jsonFile)

	err = json.Unmarshal(byteValue, def)
	if err != nil {
		fmt.Println("error:", err)
	}
}
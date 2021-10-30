package france

import (
	_ "context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	_ "log"

	_ "net"
	_"net/http"
	_ "net/http/cookiejar"
	_ "net/url"
	"os"
	_ "reflect"
	_ "strings"
	_"sync"

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
	notams := notam.NewNotamList() 	

	//Initiate a new mongo db interface
	mongoClient = database.NewMongoDb(def.CountryCode)
	aisClient = webclient.NewAisWebClient()

	def.RetrieveAllNotams()

	fmt.Printf("Current NOTAMs: %d \n", len(notams.Data))
	//Once all the NOTAM havebeen identified, identify the deleted ones and set them in the db.
	canceledNotams := mongoClient.IdentifyCanceledNotams(notams.Data)
	fmt.Printf("Canceled NOTAM: %d \n", len(*canceledNotams))
	mongoClient.SetCanceledNotamList(canceledNotams)
	mongoClient.WriteActiveNotamToFile("./web/notams/japan.json")
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
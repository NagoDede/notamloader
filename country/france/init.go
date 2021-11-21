package france

import (
	_ "context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	_ "log"
	_ "net"
	_ "net/http"
	_ "net/http/cookiejar"
	_ "net/url"
	"os"
	_ "reflect"
	_ "strings"
	"sync"
	_ "sync"

	"github.com/NagoDede/notamloader/database"
	"github.com/NagoDede/notamloader/webclient"
	_ "github.com/NagoDede/notamloader/webclient"
	_ "go.mongodb.org/mongo-driver/mongo"
	_ "golang.org/x/net/publicsuffix"
)

var FranceAis DefData

// FrData contains all the information required to connect and retrieve NOTAM from AIS services
type DefData struct {
	NotamFirRequestUrl string              `json:"notamFirRequestUrl"`
	NotamAptRequestUrl string              `json:"notamAptRequestUrl"`
	CodeListPath       string              `json:"codeListPath"`
	RequiredFirLocations  map[string][]string `json:"requiredLocation"`
	Country            string              `json:"country"`
}

var mongoClient *database.Mongodb
var aisClient *webclient.AisWebClient

// Process launches the global process to recover the NOTAMs from the Japan AIS webpages.
// It recovers the relevant information from a json file, set in ./country/japan/def.json.
// Then, it initiates the http and mongodb interfaces.
// Once achieved, it interrogates the web form by providing the location ICAO code to
// the webform to identify the reference list of the relevant NOTAM.
func (def *DefData) Process(wg *sync.WaitGroup) {

	defer wg.Done()

	//retrieve the configuration data from the json file
	def.loadJsonFile("./country/france/def.json")
	aisClient = webclient.NewAisWebClient()
	fmt.Println("Connected to AIS France")

	for afs, _ := range def.RequiredFirLocations {

		//Init a the http client thanks tp the configuration data
		//Initiate a new mongo db interface
		mongoClient = database.NewMongoDb(afs)
		//Will contain all the retrieved Notams

		retrievedNotamList := def.RetrieveAllNotams(afs)
		realNotamsList := retrievedNotamList.SendToDatabase(mongoClient)
		fmt.Printf("Applicable NOTAM: %d \n", len(realNotamsList.Data))
		canceledNotams := mongoClient.IdentifyCanceledNotams(realNotamsList.Data)
		fmt.Printf("Canceled NOTAM: %d \n", len(*canceledNotams))
		mongoClient.SetCanceledNotamList(canceledNotams)
		mongoClient.WriteActiveNotamToFile(def.Country, afs)
	}
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

package asecna

import (
	"encoding/json"
	"io/ioutil"

	"os"

	"sync"

	"github.com/NagoDede/notamloader/database"
	"github.com/NagoDede/notamloader/webclient"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// FrData contains all the information required to connect and retrieve NOTAM from AIS services
type DefData struct {
	NotamRequestUrl   string              `json:"notamRequestUrl"`
	CodeListPath      string              `json:"codeListPath"`
	RequiredLocations map[string][]string `json:"requiredLocation"`
	Country           string              `json:"country"`
}

var mongoClient *database.Mongodb
var aisClient *webclient.AisWebClient

var Logger zerolog.Logger

// Process launches the global process to recover the NOTAMs from the Japan AIS webpages.
// It recovers the relevant information from a json file, set in ./country/japan/def.json.
// Then, it initiates the http and mongodb interfaces.
// Once achieved, it interrogates the web form by providing the location ICAO code to
// the webform to identify the reference list of the relevant NOTAM.
func (def *DefData) Process(wg *sync.WaitGroup) {
	defer wg.Done()
	//retrieve the configuration data from the json file
	def.loadJsonFile("./country/asecna/def.json")
	Logger = log.With().Str("Country", def.Country).Logger()
	//Init a the http client thanks tp the configuration data
	//Initiate a new mongo db interface
	aisClient = webclient.NewAisWebClient()
	Logger.Info().Msg("Connected to AIS ASECNA")
	templog := Logger

	for afs := range def.RequiredLocations {
		Logger = templog.With().Str("AFS", afs).Logger()
		mongoClient = database.NewMongoDb(afs)
		retrievedNotamList := def.RetrieveAllNotams(afs)
		realNotamsList := mongoClient.SendToDatabase(retrievedNotamList)
		Logger.Info().Msgf("Applicable NOTAM: %d", len(realNotamsList.Data))
		canceledNotams := mongoClient.IdentifyCanceledNotams(realNotamsList.Data)
		Logger.Info().Msgf("Canceled NOTAM: %d", len(*canceledNotams))
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
		Logger.Fatal().Err(err)
	}

	// defer the closing of our jsonFile so that we can parse it later on
	defer jsonFile.Close()

	byteValue, err := ioutil.ReadAll(jsonFile)

	err = json.Unmarshal(byteValue, def)
	if err != nil {
		Logger.Fatal().Err(err)
	}
}

package canada

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
	_ "strconv"
	_ "strings"
	"sync"
	_ "sync"

	"github.com/NagoDede/notamloader/database"
	"github.com/NagoDede/notamloader/notam"
	"github.com/NagoDede/notamloader/webclient"
	_ "github.com/NagoDede/notamloader/webclient"

	_ "go.mongodb.org/mongo-driver/mongo"
	_ "golang.org/x/net/publicsuffix"

	"github.com/TwiN/go-color"
)

// FrData contains all the information required to connect and retrieve NOTAM from AIS services
type DefData struct {
	Country           string              `json:"country"`
	NotamRequestUrl   string              `json:"notamRequestUrl"`
	CodeListPath      string              `json:"codeListPath"`
	RequiredLocations map[string][]string `json:"requiredLocation"`
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
	def.loadJsonFile("./country/canada/def.json")
	//Init a the http client thanks tp the configuration data
	//Initiate a new mongo db interface
	aisClient = webclient.NewAisWebClient()
	fmt.Printf(color.Ize(color.Green,"Ready to retrieve data for %s \n"), def.Country)

	resp, err := aisClient.Get(def.NotamRequestUrl)
	if err != nil {
		fmt.Println(err)
	}
	defer resp.Body.Close()

	byteValue, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println(err)
	}

	var jsonAnswer InitJson
	err = json.Unmarshal(byteValue, &jsonAnswer)
	if err != nil {
		fmt.Println(err)
	}

	type textStruct struct {
		Raw     string `json:"raw"`
		English string `json:"english"`
		French  string `json:"french"`
	}

	notamList := notam.NewNotamList()

	for afs, _ := range def.RequiredLocations {
		mongoClient = database.NewMongoDb(afs)

		for _, notamTxt := range jsonAnswer.Data {
			var text textStruct
			err = json.Unmarshal([]byte(notamTxt.Text), &text)
			if err != nil {
				fmt.Println("Error in umarshall Canada data")
			}
			ntm := notam.NewNotamAdvanced()
			ntm.AfsCode = afs
			ntm = notam.FillNotamFromText(ntm, text.Raw)
			_, ok := notamList.Data[ntm.Id]
			if !ok {
				notamList.Data[ntm.Id] = &ntm.Notam
			}
		}

		realNotamsList := mongoClient.SendToDatabase(notamList)
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

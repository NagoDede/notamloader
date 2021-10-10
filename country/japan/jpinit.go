package japan

import (
	_ "context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"reflect"
	"strings"
	"time"

	"github.com/NagoDede/notamloader/database"
	"github.com/NagoDede/notamloader/notam"
	"golang.org/x/net/publicsuffix"
)

var JapanAis JpData

// JpLoginFormData contains relevant data to connect to the Japan AIS webform
type jpLoginFormData struct {
	FormName   string `json:"formName"`
	PasswordIn string `json:"password"`
	UserIDIn   string `json:"userID"`
	Password   string `json:"-"`
	UserID     string `json:"-"`
}

// JpData contains all the information required to connect and retrieve NOTAM from AIS services
type JpData struct {
	WebConfig
	CodeListPath     string          `json:"codeListPath"`
	codeList         jpCodeFile      //map[string]interface{}
	LoginData        jpLoginFormData `json:"loginData"`
	RequiredLocation []string        `json:"requiredLocation"`
}

// jpCodeFile is the template to retrieve the airports data from the definition files
type jpCodeFile struct {
	IsActive      bool         `json:"IsActive"`
	EffectiveDate string       `json:"EffectiveDate"`
	CountryCode   string       `json:"CountryCode"`
	Airports      []jpAirports `json:"Airports"`
}

// jpAirports is the template to extract airports information from the definition files
type jpAirports struct {
	Icao  string `json:"Icao"`
	Title string `json:"Title"`
}

// Process launches the global process to recover the NOTAMs from the Japan AIS webpages.
// It recovers the relevant information from a json file, set in ./country/japan/def.json.
// Then, it initiates the http and mongodb interfaces.
// Once achieved, it interrogates the web form by providing the location ICAO code to
// the webform to identify the reference list of the relevant NOTAM.
func (jpd *JpData) Process() {

	//retrieve the configuration data from the json file
	jpd.loadJsonFile("./country/japan/def.json")
	//Init a the http client thanks tp the configuration data
	httpClient := jpd.initClient()
	fmt.Println("Connected to AIS Japan")

	//define a default search to fullfill the form
	//Use 24h duration, retrieve the advisory and warning notams
	notamSearch := JpNotamSearchForm{
		location:   "RJNA",
		notamKbn:   "",
		period:     "24",
		dispScopeA: "true",
		dispScopeE: "true",
		dispScopeW: "true",
		firstFlg:   "true",
	}

	//ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	//defer cancel()

	//Initiate a new mongo db interface
	client := database.NewMongoDb()
	//activeNotams := client.RetrieveActiveNotams()

	var identifiedNotams []notam.NotamReference

	rand.Seed(time.Now().UnixNano())

	//Identify the NOTAM associated to an ICAO code (usually associated to an airport)
	for _, apt := range jpd.codeList.Airports {
		fmt.Printf("Retrieving NOTAM for %s \n", apt.Icao)
		notamSearch.location = apt.Icao
		//retrieve all the NOTAM references associated to the ICAO code
		//func() {
		notamReferences := notamSearch.ListNotamReferences(httpClient, jpd.WebConfig.NotamFirstPage, jpd.WebConfig.NotamNextPage)
		fmt.Printf("\t %d NOTAM reference(s) identified \n", len(notamReferences))

		//thanks the NOTAM reference, we gather the NOTAM information from the NotamDeatilPage
		for _, notamRef := range notamReferences {
			//extract the data from the webpage
			go func(ref JpNotamDispForm) {
				fmt.Printf("Ask data for %s - %s \n", ref.location, ref.notam_no)
				time.Sleep(time.Duration(rand.Intn(10)) * time.Second)
				notam, err := ref.FillInformation(httpClient, jpd.WebConfig.NotamDetailPage)
				if len(notam.Text) <= 20 {
					fmt.Printf("Get %s - %s (%s) \n %s \n", notam.Icaolocation, notam.Number, notam.Identifier, notam.Text)
				} else {
					fmt.Printf("Get %s - %s (%s) \n %s \n", notam.Icaolocation, notam.Number, notam.Identifier, notam.Text[0:20])
				}

				//no error, log theNOTAM in the Databse if it is a new one
				if err == nil {
					if client.IsNewNotam(&notam.NotamReference) {
						client.AddNotam(notam)
						identifiedNotams = append(identifiedNotams, notam.NotamReference)
						fmt.Printf("!!!! --> Identified New Notams: %d", len(identifiedNotams))
					} else {
						fmt.Printf("Not new - %s - %s \n", notam.Icaolocation, notam.Number)
					}
				} else {
					//there is an error, reset the client and start again
					fmt.Printf("\t Error to recover NOTAM %s - %s \n", ref.location, ref.notam_no)
					httpClient = jpd.initClient()
					notam, err1 := ref.FillInformation(httpClient, jpd.WebConfig.NotamDetailPage)
					if err1 != nil {
						if client.IsNewNotam(&notam.NotamReference) {
							client.AddNotam(notam)
							identifiedNotams = append(identifiedNotams, notam.NotamReference)
						}
					} else {
						fmt.Println(err1)
					}
				}
			}(notamRef)
		}
		//}()
	}
	fmt.Printf("New NOTAM: %d \n", len(identifiedNotams))
	//Once all the NOTAM havebeen identified, identify the deleted ones and set them in the db.
	canceledNotams := client.IdentifyCanceledNotams(&identifiedNotams)
	fmt.Printf("Canceled NOTAM: %d \n", len(*canceledNotams))
	client.SetCanceledNotamList(canceledNotams)
}

func structToMap(i interface{}) (values url.Values) {
	values = url.Values{}
	iVal := reflect.ValueOf(i).Elem()
	typ := iVal.Type()
	for i := 0; i < iVal.NumField(); i++ {
		values.Set(typ.Field(i).Name, fmt.Sprint(iVal.Field(i)))
	}
	return
}

/*
Load the JSON file used for the access to the Japan AIP.
The required password can be provided by an environment variable or
directly set in the Json file.
When the environement variable is used, the password definition shall respect
the syntax "Env: ENV_VARIABLE_NAME". The function will then retrieve the content
of the environment variable ENV_VARIABLE_NAME.
If the environment variable does not exist or is empty, it generates a panic.
To define an empty password, just set Password = ""  in the Json file.
The same beahavior is extended to the User ID.
*/
func (jpd *JpData) loadJsonFile(path string) {
	// Open our jsonFile
	jsonFile, err := os.Open(path)
	// if we os.Open returns an error then handle it
	if err != nil {
		fmt.Println(err)
	}

	// defer the closing of our jsonFile so that we can parse it later on
	defer jsonFile.Close()

	byteValue, err := ioutil.ReadAll(jsonFile)

	err = json.Unmarshal(byteValue, jpd)
	if err != nil {
		fmt.Println("error:", err)
	}

	//The password may be provided by an environment variable
	if strings.HasPrefix(jpd.LoginData.PasswordIn, "Env:") {
		var s = strings.TrimPrefix(jpd.LoginData.PasswordIn, "Env:")
		s = strings.TrimSpace(s)
		jpd.LoginData.Password = os.Getenv(s)

		if jpd.LoginData.Password == "" {
			panic(fmt.Sprintf("Password Environment variable: %s  not defined\n", s))
		}
	} else {
		jpd.LoginData.Password = jpd.LoginData.PasswordIn
	}

	//The UserID may be provided by an environment variable
	if strings.HasPrefix(jpd.LoginData.UserIDIn, "Env:") {
		var s = strings.TrimPrefix(jpd.LoginData.UserIDIn, "Env:")
		s = strings.TrimSpace(s)
		jpd.LoginData.UserID = os.Getenv(s)

		if jpd.LoginData.UserID == "" {
			panic(fmt.Sprintf("User ID Environment variable: %s  not defined\n", s))
		}
	} else {
		jpd.LoginData.UserID = jpd.LoginData.UserIDIn
	}

	jpd.loadCodeList(jpd.CodeListPath)
}

func (jpd *JpData) loadCodeList(path string) {
	url := path
	fmt.Printf("Load the location codes list from %s \n", url)

	resp, err := http.Get(url)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	jsonData, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}

	// Unmarshal or Decode the JSON to the interface.
	json.Unmarshal([]byte(jsonData), &jpd.codeList)
	jpd.mergeAllLocationsCodes()

	fmt.Println(jpd.codeList)

}

func (jpd *JpData) mergeAllLocationsCodes() {
	for _, code := range jpd.RequiredLocation {
		if !jpd.IsAirportCode(code) {
			apt := jpAirports{code, ""}
			jpd.codeList.Airports = append(jpd.codeList.Airports, apt)
		}
	}
}

func (jpd *JpData) IsAirportCode(code string) bool {
	for _, apt := range jpd.codeList.Airports {
		if apt.Icao == code {
			return true
		}
	}
	return false
}

/**
 * initClient inits an http client to connect to the website  by sending the
 * data to the formular.
 */
func (jpd *JpData) initClient() http.Client {

	frmData := jpd.LoginData
	//Create a cookie Jar to manage the login cookies
	jar, err := cookiejar.New(&cookiejar.Options{PublicSuffixList: publicsuffix.List})
	if err != nil {
		log.Fatal(err)
	}

	var client = http.Client{Jar: jar}
	//login to the page
	v := url.Values{"formName": {frmData.FormName},
		"password": {frmData.Password},
		"userID":   {frmData.UserID}}

	//connect to the website
	resp, err := client.PostForm(jpd.WebConfig.LoginPage, v)
	if err != nil {
		log.Printf("%s \n If error due to certificate problem, install ca-certificates", v)
		log.Fatal(err)
	}

	defer resp.Body.Close()
	return client
}

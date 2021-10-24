package japan

import (
	_ "context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"reflect"
	"strings"
	"sync"
	_ "sync"
	"time"

	"github.com/NagoDede/notamloader/database"
	"github.com/NagoDede/notamloader/notam"
	_ "go.mongodb.org/mongo-driver/mongo"
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
	CountryCode      string          `json:"countryCode"`
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
	jpd.newHttpClient()
	jpd.initHttpClient()

	fmt.Println("Connected to AIS Japan")

	//Will contain all the retrieved Notams
	allRetrievedNotams := make(map[string]notam.NotamReference)

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

	mapsearch := JpNotamMapSubmit{
		Enroute:    "1",
		Period:     "24",
		DispScopeE: "true",
		DispScopeW: "true",
	}

	//Initiate a new mongo db interface
	mongoClient := database.NewMongoDb(jpd.CountryCode)
	rand.Seed(time.Now().UnixNano())

	//Identify the NOTAM associated to an ICAO code (usually associated to an airport)
	mainStart := time.Now()
	for i, apt := range jpd.codeList.Airports {
		start := time.Now()
		fmt.Printf("Retrieving NOTAM for %s %d/%d \n", apt.Icao, i, len(jpd.codeList.Airports))
		//retrieve all the NOTAM references associated to the ICAO code
		notamSearch.location = apt.Icao
		aptNotam := notamSearch.ListNotamReferences(jpd.httpClient, jpd.WebConfig.NotamFirstPage, jpd.WebConfig.NotamNextPage)
		fmt.Printf("\t %d NOTAM reference(s) identified \n", len(aptNotam))
		//thanks the NOTAM reference, we gather the NOTAM information from the NotamDeatilPage
		jpd.getFullNotams(aptNotam, allRetrievedNotams, mongoClient, jpd.httpClient)

		if !jpd.resetOnDemand(mainStart, 10*time.Minute) {
			jpd.resetOnDemand(start, 3*time.Minute)
		}
	}

	//Retrieve the en Route NOTAMs
	//A hge amount of notam can be retrieved, to avoid server saturation,
	// do it after airport notams update.
	fmt.Printf("\n ************************ \n")
	fmt.Printf("\n Retreive En Route NOTAMs \n")
	jpd.logoutHttpClient()
	jpd.initHttpClient()
	enRouteNotamRef := mapsearch.ListNotamReferences(jpd.httpClient, jpd.WebConfig.MapPage, jpd.WebConfig.MapAnswerPage)
	jpd.logoutHttpClient()
	jpd.initHttpClient()
	jpd.getFullNotams(enRouteNotamRef, allRetrievedNotams, mongoClient, jpd.httpClient)

	defer fmt.Printf("Current NOTAMs: %d \n", len(allRetrievedNotams))
	//Once all the NOTAM havebeen identified, identify the deleted ones and set them in the db.
	canceledNotams := mongoClient.IdentifyCanceledNotams(allRetrievedNotams)
	defer fmt.Printf("Canceled NOTAM: %d \n", len(*canceledNotams))
	defer mongoClient.SetCanceledNotamList(canceledNotams)
	defer mongoClient.WriteActiveNotamToFile("./web/notams/japan.json")
}

func (jpd *JpData) getFullNotams(notamReferences []JpNotamDispForm,
	allRetrievedNotams map[string]notam.NotamReference,
	mongoClient *database.Mongodb,
	httpClient http.Client) {

	wg := new(sync.WaitGroup)
	for _, notamRef := range notamReferences {

		//skip empty items
		if (notamRef.location == "") || (notamRef.notam_no == "") {
			continue
		}

		//Record the refrence to identify the canceled
		retrievedNotam := notam.NotamReference{Number: notamRef.Number(), Icaolocation: notamRef.location, CountryCode: jpd.CountryCode}
		_, exists := allRetrievedNotams[retrievedNotam.GetKey()]
		if !exists {
			allRetrievedNotams[retrievedNotam.GetKey()] = retrievedNotam
		} else {
			//skip the next of the work. There is no need to get the data
			fmt.Printf("\t skip %s \n", retrievedNotam.GetKey())
			continue
		}

		fmt.Printf("\t Total Retrieved NOTAM %d \n", len(allRetrievedNotams))

		//extract the data from the webpage
		if !mongoClient.IsOldNotam(notamRef.GetKey()) {
			go func(wg *sync.WaitGroup, ref JpNotamDispForm) {
				wg.Add(1)
				defer wg.Done()
				fmt.Printf("(New) %s - %s \n", ref.location, ref.Number())
				time.Sleep(time.Duration(rand.Intn(10)) * time.Second)
				notam, err := ref.FillInformation(httpClient, jpd.WebConfig.NotamDetailPage, jpd.CountryCode)
				//no error, log the NOTAM in the Database if it is a new one
				if err == nil {
					if len(notam.Text) <= 20 {
						fmt.Printf("\t --> Get %s - %s (%s) \n %s \n", notam.Icaolocation, notam.Number, notam.Identifier, notam.Text)
					} else {
						fmt.Printf("\t --> Get %s - %s (%s) \n %s \n", notam.Icaolocation, notam.Number, notam.Identifier, notam.Text[0:20])
					}

					if !mongoClient.IsOldNotam(notam.NotamReference.GetKey()) {
						mongoClient.AddNotam(notam)
						fmt.Printf("\t --> Added NOTAM to db  %s - %s \n", notam.Icaolocation, notam.Number)
					} else {
						fmt.Printf("\t Not new - %s - %s \n", notam.Icaolocation, notam.Number)
					}
				} else {
					//there is an error, reset the client and start again
					fmt.Printf("\t Error to recover NOTAM %s - %s \n", ref.location, ref.notam_no)
					jpd.logoutHttpClient()
					jpd.initHttpClient()
					notam, err1 := ref.FillInformation(httpClient, jpd.WebConfig.NotamDetailPage, jpd.CountryCode)
					if err1 == nil {
						if !mongoClient.IsOldNotam(notam.NotamReference.GetKey()) {
							mongoClient.AddNotam(notam)
						}
					} else {
						fmt.Println(err1)
					}
				}

			}(wg, notamRef)
		} else {
			fmt.Printf("(Old)  %s - %s \n", notamRef.location, notamRef.Number())
		}
	}
	wg.Wait()
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

func (jpd *JpData) newHttpClient() {

	//Create a cookie Jar to manage the login cookies
	jar, err := cookiejar.New(&cookiejar.Options{PublicSuffixList: publicsuffix.List})
	if err != nil {
		log.Fatal(err)
	}

	jpd.httpClient = http.Client{Jar: jar,
		Transport: &http.Transport{
			Dial: (&net.Dialer{
				Timeout:   60 * time.Second,
				KeepAlive: 60 * time.Second,
			}).Dial,
			TLSHandshakeTimeout:   60 * time.Second,
			ResponseHeaderTimeout: 60 * time.Second,
			ExpectContinueTimeout: 20 * time.Second,
			MaxIdleConns:          0,
			MaxConnsPerHost:       0,
			MaxIdleConnsPerHost:   100,
		},
	}
}

/**
 * initClient inits an http client to connect to the website  by sending the
 * data to the formular.
 */
func (jpd *JpData) initHttpClient() {
	frmData := jpd.LoginData

	//login to the page
	v := url.Values{"formName": {frmData.FormName},
		"password": {frmData.Password},
		"userID":   {frmData.UserID}}

	//connect to the website
	resp, err := jpd.httpClient.PostForm(jpd.WebConfig.LoginPage, v)
	if err != nil {
		log.Printf("Connection Error \n If error due to certificate problem, install ca-certificates")
		log.Fatal(err)
	}
	jpd.IsConnected = true
	defer resp.Body.Close()
	jpd.httpClient.CloseIdleConnections()
}

func (jpd *JpData) logoutHttpClient() {
	if jpd.IsConnected {
		resp, err := jpd.httpClient.Get(jpd.WebConfig.LogOutPage)
		if err == nil {
			if (resp.StatusCode == 302) || (resp.StatusCode == 200) {
				fmt.Println(" Connection Logout Succes")
				jpd.IsConnected = false
				return
			}
		} else {
			fmt.Println(" Connection Logout Failed")
		}
	}
}

func (jpd *JpData) resetOnDemand(start time.Time, resetTime time.Duration) bool {

	if time.Since(start) > resetTime {
		fmt.Println("Exceed the " + resetTime.String() + " Duration -- reset Connection")
		jpd.logoutHttpClient()
		time.Sleep(time.Duration(rand.Intn(10)) * time.Second)
		jpd.initHttpClient()
		return true
	}
	return false
}

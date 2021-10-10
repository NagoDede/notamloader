package japan

import (
	_ "context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"reflect"
	"strings"
	_ "time"

	"github.com/NagoDede/notamloader/database"
	"github.com/NagoDede/notamloader/notam"
	"golang.org/x/net/publicsuffix"
)

var JapanAis JpData

type JpLoginFormData struct {
	FormName   string `json:"formName"`
	PasswordIn string `json:"password"`
	UserIDIn   string `json:"userID"`
	Password   string `json:"-"`
	UserID     string `json:"-"`
}

type JpData struct {
	WebConfig
	CodeListPath     string          `json:"codeListPath"`
	CodeList         JpCodeFile      //map[string]interface{}
	LoginData        JpLoginFormData `json:"loginData"`
	RequiredLocation []string        `json:"requiredLocation"`
}

type JpCodeFile struct {
	IsActive      bool         `json:"IsActive"`
	EffectiveDate string       `json:"EffectiveDate"`
	CountryCode   string       `json:"CountryCode"`
	Airports      []JpAirports `json:"Airports"`
}

type JpAirports struct {
	Icao  string `json:"Icao"`
	Title string `json:"Title"`
}

type WebConfig struct {
	CountryDir      string `json:"country"`
	LoginPage       string `json:"loginPage"`
	NotamFirstPage  string `json:"notamFirstPage"`
	NotamDetailPage string `json:"notamDetailPage"`
	NotamNextPage   string `json:"notamNextPage"`
}

func (jpd *JpData) Process() {

	jpd.LoadJsonFile("./country/japan/def.json")
	httpClient := jpd.InitClient()
	fmt.Println("Connected to AIS Japan")

	//define a default NOTAM
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

	client := database.NewMongoDb()
	//activeNotams := client.RetrieveActiveNotams()
	var identifiedNotams []notam.NotamReference
	for _, apt := range jpd.CodeList.Airports {
		fmt.Printf("Retrieving NOTAM for %s \n", apt.Icao)
		notamSearch.location = apt.Icao
		notamReferences := notamSearch.ListNotamReferences(httpClient, jpd.WebConfig.NotamFirstPage, jpd.WebConfig.NotamNextPage)
		fmt.Printf("\t Retrieve %d \n", len(notamReferences))

		for _, notamRef := range notamReferences {
			notam, err := notamRef.FillInformation(httpClient, jpd.WebConfig.NotamDetailPage)
			//fmt.Println(notam)
			if err == nil {
				if client.IsNewNotam(&notam.NotamReference) {
					client.AddNotam(notam)
					identifiedNotams = append(identifiedNotams, notam.NotamReference)
				}
			} else {
				httpClient = jpd.InitClient()
				notam, err1 := notamRef.FillInformation(httpClient, jpd.WebConfig.NotamDetailPage)
				if err1 != nil {
					if client.IsNewNotam(&notam.NotamReference) {
						client.AddNotam(notam)
						identifiedNotams = append(identifiedNotams, notam.NotamReference)
					}
				} else {
					fmt.Println(err1)
				}
			}
		}
	}

	canceledNotams := client.IdentifyCanceledNotams(&identifiedNotams)
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
func (jpd *JpData) LoadJsonFile(path string) {
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
	json.Unmarshal([]byte(jsonData), &jpd.CodeList)
	jpd.mergeAllLocationsCodes()

	fmt.Println(jpd.CodeList)

}

func (jpd *JpData) mergeAllLocationsCodes() {
	for _, code := range jpd.RequiredLocation {
		if !jpd.IsAirportCode(code) {
			apt := JpAirports{code, ""}
			jpd.CodeList.Airports = append(jpd.CodeList.Airports, apt)
		}
	}
}

func (jpd *JpData) IsAirportCode(code string) bool {
	for _, apt := range jpd.CodeList.Airports {
			if (apt.Icao == code) {
				return true
			}
	}
	return false
}


/**
 * initClient inits an http client to connect to the website  by sending the
 * data to the formular.
 */
func (jpd *JpData) InitClient() http.Client {

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
		log.Println("If error due to certificate problem, install ca-certificates")
		log.Fatal(err)
	}

	defer resp.Body.Close()
	return client
}

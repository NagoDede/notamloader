package japan

import (
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

	"time"

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
	CodeListPath string `json:"codeListPath"`
	CodeList     map[string]interface{}
	LoginData    JpLoginFormData `json:"loginData"`
}

func (jpd *JpData) Process() {
	startTime := time.Now()
	jpd.LoadJsonFile("./country/japan/def.json")
	jpd.Login()

	mongoClient := database.NewMongoDb()

	var identifiedNotams []notam.NotamReference
	var newNotams []*notam.NotamReference
	var newNotamCount = 0
	for code := range jpd.CodeList {

		fmt.Printf("Retrieve NOTAMs for %s \n", code)
		searchRequest := CreateSearchRequest(code)
		notamReferences := searchRequest.ListNotamReferences(&jpd.WebConfig)
		fmt.Printf("\t Retrieve %d NOTAMs for %s \n", len(notamReferences), code)

		for _, notamRef := range notamReferences {
			identifiedNotams = append(identifiedNotams, notamRef.NotamReference())
			if !mongoClient.IsNewNotam(notamRef.NotamReference()) {
				notam := notamRef.RetrieveNotam(jpd.WebConfig.httpClient, jpd.WebConfig.NotamDetailPage)
				mongoClient.AddNotam(notam)
				newNotamCount++
				tp := &identifiedNotams[len(identifiedNotams)-1]
				newNotams = append(newNotams, tp)
				fmt.Printf("\t --> %+v \n", notam.NotamReference)
			} else {
				fmt.Printf("\t %+v \n", notamRef.NotamReference())
			}

		}
	}

	canceledNotams := mongoClient.IdentifyCanceledNotams(&identifiedNotams)
	mongoClient.SetCanceledNotamList(canceledNotams)

	fmt.Println("** Report: ")
	fmt.Printf("\t Identifed Notams: %d \n", len(identifiedNotams))
	fmt.Printf("\t New Notams: %d \n", newNotamCount)
	for _, ntm := range newNotams {
		fmt.Printf(" %+v", *ntm)
	}

	fmt.Printf("\t Canceled Notams: %d \n", len(*canceledNotams))

	elapsed := time.Since(startTime)
	log.Printf("Japan Notams tooks %s", elapsed)
}

func structToUrlValues(i interface{}) (values url.Values) {
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

	json.Unmarshal([]byte(jsonData), &jpd.CodeList)

	fmt.Printf("%d codes retrieved", len(jpd.CodeList))
}

/**
 * Login inits an http client to connect to the website  by sending the
 * data to the formular.
 */
func (jpd *JpData) Login() {
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

	//TODO confirm it is well connected
	fmt.Println("Connected to AIS Japan")

	defer resp.Body.Close()
	jpd.WebConfig.httpClient = client

}

package webclient

import (
	"encoding/json"

	"io/ioutil"

	"net/http"
)

type AirportData struct{
	Icao string `json:"icao"`
	Iata string `json:"iata"`
	Name string `json:"name"`
	Location string `json:"location"`
	Country string `json:"country"`
	CountryCode string `json:"country_code"`
	Longitude string  `json:"longitude"`
	Latitude string `json:"latitude"`
	Link	string `json:"link"`
	Status int `json:"status"`
	}

	//Retrieve from Airport-Data.com the airport information
func GetAirportData(icao string) *AirportData{
	body, err := http.Get("https://www.airport-data.com/api/ap_info.json?icao=" + icao)
	if err != nil{
		return nil
	}

	byteValue, err := ioutil.ReadAll(body.Body)
	var aptdata AirportData
	json.Unmarshal(byteValue, &aptdata)
	
	return &aptdata
}
package canada

type InitJson struct{
	Meta struct {
		Now 	string		`json:"now"`
		Count struct {
			NotamCount uint32	`json:"notam"`
		} `json:"count"`
		Messages []string	`json:"messages"`
	}	`json:"meta"`
	Data 	[]DataStruct	`json:"data"`
}

type DataStruct struct{
	Type 			string `json:"type"`
	Pk 				string 		`json:"pk"`
	Location 		string 	`json:"location"`
	StartValidity	string	`json:"startValidity"`
	EndValidity		string	`json:"endValidity"`
	Text			string	`json:"text"`
	HasError		bool	`json:"hasError"`
	Position struct {
		PointReference	string `json:"pointReference"`
		RadialDistance	uint32 `json:"radialDistance"`
	}	`json:"position"`
}

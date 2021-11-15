package notam

import (
	"fmt"
	"math"
	"time"
)

// Refer to ICAO Annex 15 Appendix 6
// A NOTAM date-time group use a ten-figure group, 
// giving year, month, day, hours and minutes in UTC. 
// This entry is the date-time at which the NOTAMN comes into force. 
// In the cases of NOTAMR and NOTAMC, the date-time group is 
// the actual date and time of the NOTAM origination. 
// The start of a day shall be indicated by “0000”. 
// The end of a day shall be indicated by “2359”.
// With the exception of NOTAMC, a date-time group indicating duration 
// of information shall be used unless the information is of a 
// permanent nature in which case the abbreviation “PERM” is inserted instead. 
// If information on timing is uncertain, the approximate duration -Item C) - shall be 
// indicated using a date-time group followed by the abbreviation “EST”. 
// Any NOTAM which includes an “EST” shall be cancelled or replaced before 
// the date-time specified in Item C).
const (
	NotamDateLayout string = "0601021504"
)

// Converts the NOTAM date (yymmddhhmm) to date
// The Golang date parse is limited by the use of two-digit year.
// The function overcomes the limitation thanks the recognition of the current year.
func NotamDateToTime(ndte string, loc *time.Location) (time.Time, error) {
	
	//utc := time.UTC
	parsedate, err := time.ParseInLocation(NotamDateLayout, ndte, loc)
	// For layouts specifying the two-digit year 06, a value NN >= 69 will be treated as 19NN and a value NN < 69 will be treated as 20NN.
	if parsedate.Year() < time.Now().Year() {
		var mil float64
		//retrieve the first two-digit of the current year
		mil = float64(time.Now().Year() / 100.0)
		mil, _ = math.Modf(mil)
		// complete the notam year to get a four-digit year
		ndte = fmt.Sprintf("%d%s", int(mil), ndte)
		layout := "200601021504"
		parsedate, err = time.ParseInLocation(layout, ndte, loc)
	}

	return parsedate, err
}

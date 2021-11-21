package notam

import (
	"fmt"
	"math"
	"strings"

	"time"

	"github.com/rs/zerolog/log"
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

// Nevertheless, all countries do not respect the standard. The list below
// has to be completed when a new layout is identified
var datesLayouts = map[string]string{
	"Icao": "0601021504",
	//Asecna has sometimes different layout
	"Asecna": "2006-01-02 15:04:05",
	//France has a different format.
	"France": "2006 Jan 02 15:04",
	//Mostly to complete the icao date
	"Default": "200601021504",
	//Local date are not present, but sure... a guy will use them
	"IcaoLocal": "0601021504 MST",
	//To generate lisible data
	"Text": "Mon Jan 2 15:04 UTC 2006",
}

var textLayout string = datesLayouts["Text"]

// Converts the NOTAM date (yymmddhhmm) to date
// The Golang date parse is limited by the use of two-digit year.
// The function overcomes the limitation thanks the recognition of the current year.
func NotamDateToTime(ndte string, layout string, loc *time.Location) (time.Time, error) {
	var parsedate time.Time
	var err error
	if loc == nil {
		parsedate, err = time.Parse(layout, ndte)
	} else {
		parsedate, err = time.ParseInLocation(layout, ndte, loc)

	}
	// For layouts specifying the two-digit year 06, a value NN >= 69 will be treated as 19NN and a value NN < 69 will be treated as 20NN.
	if !strings.Contains(layout, "2006") &&
		parsedate.Year() < time.Now().Year() {
		var mil float64
		//retrieve the first two-digit of the current year
		mil = float64(time.Now().Year() / 100.0)
		mil, _ = math.Modf(mil)
		// complete the notam year to get a four-digit year
		ndte = fmt.Sprintf("%d%s", int(mil), ndte)
		//layout := "200601021504"
		parsedate, err = time.ParseInLocation(datesLayouts["Default"], ndte, loc)
	}

	//log.Trace().Msgf("Parse date %s to %s with %s %s", ndte, parsedate.Format(layout), layout, loc)

	return parsedate, err
}

// Fill the fields FromDateUtCTime and FromDateUtcClear of an advanced notam.
// This is done in accordance with the layout defined by the DateFromFormat.
// If the layout is not formally identified, it tries to identify if it's an
// other format.
func (ntm *NotamAdvanced) fillFromDates() {

	for _, layout := range datesLayouts {

		if len(ntm.FromDate) == len(layout) {
			var parsed time.Time
			var err error
			if strings.Contains(layout, "MST") {
				parsed, err = NotamDateToTime(ntm.FromDate, layout, nil)
				if err == nil {
					ntm.FromDateUtcTime = parsed
					ntm.FromDateUtcClear = parsed.Format(textLayout)
					return
				}
			} else {
				parsed, err = NotamDateToTime(ntm.FromDate, layout, time.UTC)
				if err == nil {
					ntm.FromDateUtcTime = parsed
					ntm.FromDateUtcClear = parsed.Format(textLayout)
					return
				}
			}
		}
	}

	//clearly we have a problem to read the date
	log.Warn().Msgf("Notam: %s - %s not a valid date FROM identified", ntm.Number, ntm.FromDate)

}

func (ntm *NotamAdvanced) fillToDates() {

	var isEstimate bool = false
	var tmpDate string = ntm.ToDate

	if ntm.ToDate == "PERM" {
		ntm.ToDateUtcClear = "Permanent"
		log.Trace().Msgf("Notam: %s - PERMANENT NOTAM", ntm.Number)
		return
	}

	if strings.Contains(ntm.ToDate, "EST") {
		ntm.ToDateUtcClear = "Estimate"
		isEstimate = true
		//remove the EST to simplify date work
		tmpDate = strings.ReplaceAll(tmpDate, "EST", "")
		tmpDate = strings.Trim(tmpDate, " \n\r\t")
	}

	for _, layout := range datesLayouts {

		if len(tmpDate) == len(layout) {
			var parsed time.Time
			var err error
			if strings.Contains(layout, "MST") {
				parsed, err = NotamDateToTime(tmpDate, layout, nil)
				if err == nil {
					ntm.ToDateUtcTime = parsed
					ntm.ToDateUtcClear = parsed.Format(textLayout)
					if isEstimate {
						ntm.ToDateUtcClear = ntm.ToDateUtcClear + " Estimated"
					}
					return
				}
			} else {
				parsed, err = NotamDateToTime(tmpDate, layout, time.UTC)
				if err == nil {
					ntm.ToDateUtcTime = parsed
					ntm.ToDateUtcClear = parsed.Format(textLayout)
					if isEstimate {
						ntm.ToDateUtcClear = ntm.ToDateUtcClear + " Estimated"
					}
					return
				}
			}
		}
	}

	
	//clearly we have a problem to read the date
	log.Warn().Msgf("Notam: %s - %s(%d) not a valid date TO",
		ntm.Number,
		ntm.ToDate,
		len(ntm.ToDate))
}

package notam

import (
	"testing"
	"time"
)

func TestNotamDateToTime(t *testing.T) {
	loc, _ := time.LoadLocation("UTC")
	dates := [4]string{"0601021504", "6901021504", "9912312359", "0001010000"}
	realdates := [4]time.Time{
		time.Date(2006, 1, 2, 15, 04, 00, 0, loc),
		time.Date(2069, 1, 2, 15, 04, 00, 0, loc),
		time.Date(2099, 12, 31, 23, 59, 00, 0, loc),
		time.Date(2000, 1, 1, 00, 00, 00, 0, loc)}
	//For layouts specifying the two-digit year 06, a value NN >= 69 will be treated as 19NN and a value NN < 69 will be treated as 20NN.
	//Test that we will retrieve only 20NN value even if NN >=69
	for i, _ := range dates {
		got := NotamDateToTime(dates[i])

		if got != realdates[i] {
			t.Errorf("Got %s instead %s \n", got, realdates[i])
		} else {
			t.Logf("Confirm %s -> %s \n", dates[i], got)
		}
	}
}

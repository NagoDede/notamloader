package france

import (
	"fmt"
	"io"
	"log"
	"net/http"
	_"strconv"
	"time"

	_"github.com/NagoDede/notamloader/webclient"
)

func (def *DefData) RetrieveAllNotams()  {
	for _, icaoCode := range def.RequiredLocation{
		form := NewFormRequest(icaoCode, getFormatedDate(), getFormatedHour())
		resp, err := def.SendRequest(form)
		if err != nil {
			log.Fatalln(err)	
		}
		defer resp.Body.Close()
		b, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Fatalln(err)
		}

		fmt.Println(string(b))
	}
}

func (def *DefData) SendRequest(form *FormRequest) (*http.Response, error) {
	return aisClient.Client.PostForm(def.NotamRequestUrl, form.Encode())//aisClient.SendPost(def.NotamRequestUrl, form)
}

func getFormatedDate() string{
	currentDate := time.Now().UTC()
	return currentDate.Format("2006/01/02")
}

func getFormatedHour() string{
	currentUtc := time.Now().UTC().Add( 10 * time.Minute)
	return currentUtc.Format("15:04")


}
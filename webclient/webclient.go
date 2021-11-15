package webclient

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"sync"
	"time"
	"reflect"
	"golang.org/x/net/publicsuffix"
)
type AisWebClient struct {
	sync.RWMutex
	Client *http.Client
}

func NewAisWebClient() *AisWebClient {
	client := &AisWebClient{}
	client.Client = newHttpClient()
	return client
}


func  newHttpClient() *http.Client {

	//Create a cookie Jar to manage the login cookies
	jar, err := cookiejar.New(&cookiejar.Options{PublicSuffixList: publicsuffix.List})
	if err != nil {
		log.Fatal(err)
	}

	return &http.Client{Jar: jar,
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

func StructToMap(i interface{}) (values url.Values) {
	values = url.Values{}
	iVal := reflect.ValueOf(i).Elem()
	typ := iVal.Type()
	for i := 0; i < iVal.NumField(); i++ {
		
		//values.Set(typ.Field(i).Name, fmt.Sprint(iVal.Field(i)))
		
		fi := typ.Field(i)
		name := fi.Tag.Get("json")
		if name=="" {
			name = fi.Name
		}
		
		values.Set(name, fmt.Sprint(iVal.Field(i)))
		//TODO compléter si Field est un array
	}
	return
}

func (aisclient *AisWebClient) SendPost(url string, form interface{}) (resp *http.Response, err error){
	urlValues := StructToMap(form)
	aisclient.RLock()
	resp, err = aisclient.Client.PostForm(url, urlValues)
	aisclient.RUnlock()
	return
}

func (aisclient *AisWebClient) Get(url string) (resp *http.Response, err error){
	return aisclient.Client.Get(url)
}
package webclient

import (
	"log"
	"net"
	"net/http"
	"net/http/cookiejar"
	"sync"
	"time"

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
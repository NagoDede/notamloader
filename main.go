package main

import (
	"fmt"
	_ "fmt"
	"sync"
	"os"

	"github.com/NagoDede/notamloader/country/asecna"
	"github.com/NagoDede/notamloader/country/canada"
	"github.com/NagoDede/notamloader/country/france"
	_ "github.com/NagoDede/notamloader/country/japan"

    "github.com/rs/zerolog"
    "github.com/rs/zerolog/log"
)


func logInit(){
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
}

func main() {

	logInit()

	wg := new(sync.WaitGroup)
	wg.Add(3)

	canadaProcessor := canada.DefData{}
	go canadaProcessor.Process(wg)

	asecnaNotamProcessor := asecna.DefData{}
	go asecnaNotamProcessor.Process(wg)

	franceNotamProcessor := france.DefData{}
	go franceNotamProcessor.Process(wg)

	//japanNotamProcessor := japan.JpData{}
	//go japanNotamProcessor.Process(wg)

	wg.Wait()
	fmt.Println("All Process Ended")
}


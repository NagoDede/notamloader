package main

import (
	"fmt"
	_ "fmt"
	"sync"

	"github.com/NagoDede/notamloader/country/asecna"
	"github.com/NagoDede/notamloader/country/canada"
	"github.com/NagoDede/notamloader/country/france"
	_ "github.com/NagoDede/notamloader/country/japan"
)

func main() {

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

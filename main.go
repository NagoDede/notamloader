package main

import (
	_ "fmt"

	"github.com/NagoDede/notamloader/country/france"
	_ "github.com/NagoDede/notamloader/country/japan"
)

func main() {

	franceNotamProcessor := france.DefData{}
	franceNotamProcessor.Process()

	//japanNotamProcessor := japan.JpData{}
	// japanNotamProcessor.Process()
}

package main

import (
	_ "fmt"

	"github.com/NagoDede/notamloader/country/france"
	 "github.com/NagoDede/notamloader/country/japan"
)

func main() {

	franceNotamProcessor := france.DefData{}
	go franceNotamProcessor.Process()

	japanNotamProcessor := japan.JpData{}
	 japanNotamProcessor.Process()
}

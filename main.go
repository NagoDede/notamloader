package main

import (
	_ "fmt"

	_"github.com/NagoDede/notamloader/country/france"
	 "github.com/NagoDede/notamloader/country/japan"
)

func main() {

	//franceNotamProcessor := france.DefData{}
	//franceNotamProcessor.Process()

	 japanNotamProcessor := japan.JpData{}
	 japanNotamProcessor.Process()
}

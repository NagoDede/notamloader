package main

import (
	"fmt"
	"github.com/NagoDede/notamloader/country/japan"

)

func main() {

	japanNotamProcessor := japan.JpData{}
	fmt.Println("NOTAM Downloader is starting")

	japanNotamProcessor.Process()
}

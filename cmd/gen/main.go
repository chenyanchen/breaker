package main

import (
	"flag"
	"fmt"
)

func main() {
	_package := flag.String("package", "", "The packages named by the import paths")
	_interface := flag.String("interface", "", "(optional) The interfaces name")
	flag.Parse()

	fmt.Println(*_package, *_interface)

	// TODO: parse package

	// TODO: generate code
}

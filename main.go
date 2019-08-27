package main

import (
	"flag"

	"github.com/labstack/echo"
)

var (
	url = flag.String("url", ":8080", "Url to listen to")
)

func main() {
	flag.Parse()
	e := echo.New()

	e.Logger.Fatal(e.Start(*url))
}

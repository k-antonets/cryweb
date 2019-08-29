package main

import (
	"flag"

	"github.com/foolin/goview"
	"github.com/foolin/goview/supports/echoview"
	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
)

var (
	url = flag.String("url", ":8080", "Url to listen to")
)

func main() {
	flag.Parse()
	e := echo.New()
	e.Use(middleware.StaticWithConfig(middleware.StaticConfig{
		Skipper: middleware.DefaultSkipper,
		Root:    "/static",
		Index:   "index.html",
		HTML5:   true,
		Browse:  false,
	}))
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	e.Renderer = echoview.New(goview.Config{
		Root:      "templates",
		Extension: ".tmpl",
		Master:    "layouts/base",
		Partials:  []string{"assets/js", "assets/style", "assets/login"},
	})

	e.Use(middleware.BodyLimit("400M"))

	e.GET("/", IndexHndl)
	e.GET("/login", LoginHndl)

	e.Logger.Fatal(e.Start(*url))
}

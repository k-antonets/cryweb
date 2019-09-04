package main

import (
	"flag"
	"net/http"

	"github.com/foolin/goview"
	"github.com/foolin/goview/supports/echoview"
	"github.com/lab7arriam/cryweb/handlers"
	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
	"gopkg.in/go-playground/validator.v9"
	"gopkg.in/mgo.v2"
)

var (
	url    = flag.String("url", ":8080", "Url to listen to")
	mongo  = flag.String("mongo", "", "Url to mongo server")
	mdb    = flag.String("db", "cry_processor", "Database to use")
	jwtkey = flag.String("jwt", "secret", "jwt signing key")
)

func main() {
	flag.Parse()
	e := echo.New()
	e.Validator = &CustomValidator{validator: validator.New()}
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

	db, err := mgo.Dial(*mongo)
	if err != nil {
		e.Logger.Fatal(err)
	}

	h := &handlers.Handler{DB: db, Database: *mdb, Key: *jwtkey}

	e.Use(middleware.BodyLimit("400M"))

	e.Use(middleware.JWTWithConfig(middleware.JWTConfig{
		Skipper: func(c echo.Context) bool {
			if c.Path() == "/" || c.Path() == "/register" || c.Path() == "/login" {
				return true
			}
			return false
		},
		SigningKey:  []byte(h.Key),
		TokenLookup: "cookie:token",
	}))

	e.GET("/", func(c echo.Context) error {
		return c.Render(http.StatusOK, "pages/index", echo.Map{})
	})
	e.GET("/login", func(c echo.Context) error {
		return c.Render(http.StatusOK, "pages/login", echo.Map{})
	})
	e.GET("/register", func(c echo.Context) error {
		return c.Render(http.StatusOK, "pages/register", echo.Map{})
	})

	e.POST("/login", h.Login)

	e.POST("/register", h.Register)

	e.Logger.Fatal(e.Start(*url))
}

type CustomValidator struct {
	validator *validator.Validate
}

func (cv *CustomValidator) Validate(i interface{}) error {
	return cv.validator.Struct(i)
}

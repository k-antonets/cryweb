package main

import (
	"flag"
	"github.com/lab7arriam/cryweb/providers"
	"net/http"

	"github.com/foolin/goview"
	"github.com/foolin/goview/supports/echoview-v4"
	"github.com/lab7arriam/cryweb/handlers"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/spf13/viper"
	"gopkg.in/go-playground/validator.v9"
	"gopkg.in/mgo.v2"
)

var (
	config = flag.String("config", "config.cfg", "path ro config file")
)

func main() {
	flag.Parse()

	viper.SetConfigName(*config)
	viper.AddConfigPath("/run/secrets")
	viper.AddConfigPath(".")

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

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			e.Logger.Fatal("config file not found: " + err.Error())
		} else {
			e.Logger.Fatal("error with reading config file: " + err.Error())
		}
	}

	e.Renderer = echoview.New(goview.Config{
		Root:      "templates",
		Extension: ".tmpl",
		Master:    "layouts/base",
		Partials:  []string{"assets/js", "assets/style", "assets/login"},
	})

	db, err := mgo.Dial(viper.GetString("mongo"))
	if err != nil {
		e.Logger.Fatal(err)
	}

	h := &handlers.Handler{
		DB:       db,
		Database: viper.GetString("database"),
		Key:      viper.GetString("jwt_key"),
		ES: providers.NewEmailSender(providers.NewSmptClient(viper.GetString("email.server"),
			viper.GetString("email.login"),
			viper.GetString("email.password")),
			"templates/emails/*"),
		Url: viper.GetString("url"),
	}

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

	e.GET("/activate", h.Activate)

	e.POST("/login", h.Login)

	e.POST("/register", h.Register)

	e.Logger.Fatal(e.Start(viper.GetString("url")))
}

type CustomValidator struct {
	validator *validator.Validate
}

func (cv *CustomValidator) Validate(i interface{}) error {
	return cv.validator.Struct(i)
}

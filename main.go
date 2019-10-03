package main

import (
	"flag"
	"github.com/lab7arriam/cryweb/providers"
	"github.com/labstack/gommon/log"

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
	viper.AddConfigPath("/")
	viper.AddConfigPath(".")

	e := echo.New()
	e.Validator = &CustomValidator{validator: validator.New()}

	e.Pre(middleware.AddTrailingSlash())

	e.Use(middleware.StaticWithConfig(middleware.StaticConfig{
		Skipper: middleware.DefaultSkipper,
		Root:    "/static",
		Index:   "index.html",
		HTML5:   true,
		Browse:  false,
	}))
	e.Use(middleware.Logger())
	e.Logger.SetLevel(log.DEBUG)
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

	e.Logger.Info("Connecting to mongo at url: " + viper.GetString("mongo"))
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
			viper.GetString("email.name"),
			viper.GetString("email.password")),
			"templates/emails/*"),
		Url: viper.GetString("domain"),
	}

	e.GET("/", func(c echo.Context) error {
		return c.Redirect(http.StatusMovedPermanently, "/tools/cry_processor")
	})

	user := e.Group("/user")

	user.GET("/login/", func(c echo.Context) error {
		return c.Render(http.StatusOK, "pages/login", echo.Map{
			"redirect_url": c.QueryParam("redirect_url"),
		})
	})
	user.GET("/register/", func(c echo.Context) error {
		return c.Render(http.StatusOK, "pages/register", echo.Map{
			"action_url": e.Reverse("user.register"),
		})
	})

	user.GET("/activate/", h.Activate)

	user.POST("/login/", h.Login).Name = "user.login"

	user.POST("/register/", h.Register).Name = "user.register"

	tools := e.Group("/tools")
	cry_processor := tools.Group("/cry_processor")

	cry_processor.GET("/", func(c echo.Context) error {
		return c.Render(http.StatusOK, "pages/index", echo.Map{
			"login_url": e.Reverse("user.login"),
		})
	})

	tasks := cry_processor.Group("/tasks/")

	tasks.Use(middleware.BodyLimit("400M"))

	tasks.Use(middleware.JWTWithConfig(middleware.JWTConfig{
		Skipper: func(c echo.Context) bool {
			return false
		},
		SigningKey:  []byte(h.Key),
		TokenLookup: "cookie:token",
	}))

	e.Logger.Fatal(e.Start(viper.GetString("url")))
}

type CustomValidator struct {
	validator *validator.Validate
}

func (cv *CustomValidator) Validate(i interface{}) error {
	return cv.validator.Struct(i)
}

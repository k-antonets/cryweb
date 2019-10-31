package main

import (
	"flag"
	"html/template"
	"strings"

	"github.com/lab7arriam/cryweb/models"
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
		Funcs: template.FuncMap{
			"get_results_url": GetResultsResolver(e),
			"capitalize":      strings.ToTitle,
		},
	})

	e.Logger.Info("Connecting to mongo at url: " + viper.GetString("mongo"))
	db, err := mgo.Dial(viper.GetString("mongo"))
	if err != nil {
		e.Logger.Fatal(err)
	}

	e.Logger.Info("Creating celery worker client")

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
		Route: func(name string, params ...interface{}) string {
			return e.Reverse(name, params...)
		},
	}

	h.InitCelery(viper.GetString("redis_url"), viper.GetInt("workers_number"))

	e.GET("/", func(c echo.Context) error {
		return c.Redirect(http.StatusMovedPermanently, e.Reverse("tools.main", "cry_processor"))
	}).Name = "main.page"

	user := e.Group("/user")

	user.GET("/login/", func(c echo.Context) error {
		return c.Render(http.StatusOK, "pages/login", echo.Map{
			"redirect_url": c.QueryParam("redirect_url"),
			"register_url": e.Reverse("user.register"),
			"login_url":    e.Reverse("user.login"),
		})
	})
	user.GET("/register/", func(c echo.Context) error {
		return c.Render(http.StatusOK, "pages/register", echo.Map{
			"action_url": e.Reverse("user.register"),
			"cancel_url": e.Reverse("main.page"),
		})
	})

	user.GET("/activate/", h.Activate).Name = "user.activate"

	user.POST("/login/", h.Login).Name = "user.login"

	user.POST("/register/", h.Register).Name = "user.register"

	tools := e.Group("/tools/:tool")

	tools.GET("/", func(c echo.Context) error {
		if c.Param("tool") != "cry_processor" {
			return c.Render(http.StatusNotFound, "pages/index", echo.Map{
				"tool_name":    c.Param("tool"),
				"login_url":    e.Reverse("user.login"),
				"register_url": e.Reverse("user.register"),
				"notification": "Tool is not found",
				"alert_type":   "error",
			})
		}
		tool_name := "Cry Processor" // TODO: should be replaced by adding tools to db
		return c.Render(http.StatusOK, "pages/index", echo.Map{
			"tool_name":    tool_name,
			"login_url":    e.Reverse("user.login"),
			"register_url": e.Reverse("user.register"),
		})
	}).Name = "tools.main"

	tasks := tools.Group("/tasks")

	tasks.Use(middleware.BodyLimit("400M"))

	tasks.Use(middleware.JWTWithConfig(middleware.JWTConfig{
		Skipper: func(c echo.Context) bool {
			return false
		},
		Claims:      &handlers.JwtUserClaims{},
		SigningKey:  []byte(h.Key),
		TokenLookup: "cookie:token",
	}))

	tasks.GET("/", h.TasksList).Name = "tasks.list"

	tasks.GET("/add/", func(c echo.Context) error {
		if c.Param("tool") != "cry_processor" {
			return c.Redirect(http.StatusBadRequest, e.Reverse("tasks.list", c.Param("tool")))
		}
		tool_name := "Cry Processor" // TODO: should be replaced by adding tools to db
		return c.Render(http.StatusOK, "pages/add_task", echo.Map{
			"tool_name":  tool_name,
			"action_url": e.Reverse("tasks.add", c.Param("tool")),
			"cancel_url": e.Reverse("tasks.list", c.Param("tool")),
		})
	}).Name = "tasks.add_form"

	tasks.POST("/add/", h.AddTask).Name = "tasks.add"

	tasks.GET("/results/:task/", func(context echo.Context) error {
		return nil
	}).Name = "tasks.result"

	e.Logger.Fatal(e.Start(viper.GetString("url")))
}

type CustomValidator struct {
	validator *validator.Validate
}

func (cv *CustomValidator) Validate(i interface{}) error {
	return cv.validator.Struct(i)
}

func GetResultsResolver(e *echo.Echo) func(task *models.Task) string {
	return func(task *models.Task) string {
		return e.Reverse("tasks.result", task.Tool, task.Id)
	}
}

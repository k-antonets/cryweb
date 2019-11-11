package main

import (
	"flag"
	_ "github.com/dgrijalva/jwt-go"
	"html/template"
	"net/url"
	"strings"

	mw "github.com/lab7arriam/cryweb/middleware"
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

	if viper.GetBool("production") {
		e.Pre(middleware.HTTPSRedirect())
	}

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
		Partials:  []string{"assets/js", "assets/style", "assets/login", "assets/logged"},
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
		Threads: viper.GetInt("threads"),
		WorkDir: viper.GetString("workdir"),
	}

	e.Use(mw.JWTWithConfig(mw.JWTConfig{
		Skipper: func(c echo.Context) bool {
			return false
		},
		Claims:         &handlers.JwtUserClaims{},
		SigningKey:     []byte(h.Key),
		TokenLookup:    "cookie:token",
		ContextKey:     "auth",
		ContextKeyFlag: "logged",
	}))

	e.Logger.Info("Creating celery worker client")

	if err := h.InitCelery(viper.GetString("redis_url"),
		viper.GetInt("workers_number"),
		viper.GetInt("timeout"),
		viper.GetString("support_email")); err != nil {
		e.Logger.Error(err)
	}

	e.GET("/", func(c echo.Context) error {
		return c.Redirect(http.StatusMovedPermanently, e.Reverse("tools.main", "cry_processor"))
	}).Name = "main.page"

	user := e.Group("/user")

	user.GET("/login/", h.LoginPage)

	user.GET("/register/", h.RegisterPage)

	user.GET("/activate/", h.Activate).Name = "user.activate"

	user.POST("/login/", h.Login).Name = "user.login"

	user.POST("/register/", h.Register).Name = "user.register"

	tools := e.Group("/tools/:tool")

	tools.GET("/", h.ToolMain).Name = "tools.main"

	tasks := tools.Group("/tasks")

	tasks.Use(middleware.BodyLimit("400M"))

	tasks.Use(middleware.JWTWithConfig(middleware.JWTConfig{
		Skipper: func(c echo.Context) bool {
			return false
		},
		Claims:      &handlers.JwtUserClaims{},
		SigningKey:  []byte(h.Key),
		TokenLookup: "cookie:token",
		ErrorHandlerWithContext: func(err error, ctx echo.Context) error {
			redirect_url := ctx.Request().URL.String()
			login_url := e.Reverse("user.login")
			lu, err2 := url.Parse(login_url)
			if err2 != nil {
				return err2
			}
			params := url.Values{}
			params.Add("redirect_url", redirect_url)
			lu.RawQuery = params.Encode()
			return ctx.Redirect(http.StatusMovedPermanently, lu.String())
		},
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

	tasks.GET("/results/:task/", h.GetResults).Name = "tasks.result"

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
		if task.ResultExists() {
			return e.Reverse("tasks.result", task.Tool, task.Id.Hex())
		} else {
			return ""
		}
	}
}

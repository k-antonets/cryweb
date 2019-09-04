package handlers

import (
	"net/http"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/lab7arriam/cryweb/models"
	"github.com/labstack/echo"
	"gopkg.in/go-playground/validator.v9"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

func (h *Handler) Login(c echo.Context) error {
	u := new(models.User)
	if err := c.Bind(u); err != nil {
		c.Logger().Error(err)
		return indexAlerts(c, http.StatusBadRequest, "Malformed request", "danger")
	}

	if err := h.DB.DB(h.Database).
		C("users").Find(bson.M{"email": u.Email}).One(u); err != nil {
		if err == mgo.ErrNotFound {
			return indexAlerts(c, http.StatusUnauthorized, "Invalid email or password", "danger")
		}
		c.Logger().Error(err)
		return indexAlerts(c, http.StatusUnauthorized, "Invalid email or password", "danger")
	}

	pwd := c.FormValue("password")
	if !u.CheckPassword(pwd) {
		return indexAlerts(c, http.StatusUnauthorized, "Invalid email or password", "danger")
	}

	if !u.IsActive() {
		return indexAlerts(c, http.StatusForbidden, "Your account is waiting for activation", "danger")
	}

	expires := time.Hour * 24

	claims := &jwtUserClaims{
		Email: u.Email,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: time.Now().Add(expires).Unix(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	t, err := token.SignedString([]byte(h.Key))
	if err != nil {
		return err
	}

	c.SetCookie(&http.Cookie{
		Name:   "token",
		Value:  t,
		MaxAge: int(expires / time.Second),
	})
	return indexAlerts(c, http.StatusOK, "You are logged in!", "success")
}

func (h *Handler) Register(c echo.Context) error {
	u := models.NewUser()

	if err := c.Bind(u); err != nil {
		c.Logger().Error(err)
		return indexAlerts(c, http.StatusBadRequest, "Malformed request", "danger")
	}

	if err := c.Validate(u); err != nil {
		return indexAlerts(c, http.StatusBadRequest, err.Error(), "danger")
	}

	pwd1, pwd2 := c.FormValue("password"), c.FormValue("password2")

	if err := validator.New().VarWithValue(pwd1, pwd2, "required,eqfield"); err != nil {
		return indexAlerts(c, http.StatusBadRequest, "Passwords do not match", "danger")
	}

	if err := u.SetPassword(pwd1); err != nil {
		c.Logger().Error(err)
		return indexAlerts(c, http.StatusBadRequest, "Failed to register new user", "danger")
	}

	if err := h.DB.DB(h.Database).C("users").Insert(u); err != nil {
		if mgo.IsDup(err) {
			return indexAlerts(c, http.StatusForbidden, "User with the same email address already exists", "danger")
		}
		c.Logger().Error(err)
		return indexAlerts(c, http.StatusBadRequest, "Failed to register new user", "danger")
	}

	return indexAlerts(c, http.StatusOK, "You were successfully registered. Please, activate your account with the link in the email.", "success")
}

type jwtUserClaims struct {
	Email string `json:"email"`
	jwt.StandardClaims
}

func indexAlerts(ctx echo.Context, code int, notification, alert string) error {
	return ctx.Render(code, "pages/index", echo.Map{
		"notification": notification,
		"alert_type":   alert,
	})
}

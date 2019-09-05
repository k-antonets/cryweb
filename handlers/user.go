package handlers

import (
	"net/http"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/lab7arriam/cryweb/models"
	"github.com/labstack/echo/v4"
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
		C("users").FindId(u.Email).One(u); err != nil {
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

	link, err := u.GetActivationUrl(h.Key, h.Url)
	if err != nil {
		c.Logger().Error(err)
		return indexAlerts(c, http.StatusBadRequest, "Failed to register new user", "danger")
	}

	if err := h.ES.Send([]string{u.Email}, "registered", echo.Map{
		"link":    link,
		"subject": "Registration at CryProcessor web server",
	}); err != nil {
		return indexAlerts(c, http.StatusBadGateway, "Failed to sent email with activation link", "danger")
	}

	return indexAlerts(c, http.StatusOK, "You were successfully registered. Please, activate your account with the link in the email.", "success")
}

func (h *Handler) Activate(c echo.Context) error {
	email, hash, admin := c.FormValue("email"), c.FormValue("hash"), c.FormValue("admin")
	v := validator.New()

	if v.Var(email, "required,email") != nil || v.Var(hash, "required") != nil {
		return indexAlerts(c, http.StatusBadGateway, "Wrong email or hash", "danger")
	}

	u := models.NewUser()

	if err := h.DB.DB(h.Database).
		C("users").FindId(email).One(u); err != nil {
		if err == mgo.ErrNotFound {
			return indexAlerts(c, http.StatusUnauthorized, "Invalid email or password", "danger")
		}
		c.Logger().Error(err)
		return indexAlerts(c, http.StatusUnauthorized, "Invalid email or password", "danger")
	}

	if v.Var(admin, "required,email") == nil {
		if !u.IsActive() {
			uadmin := models.NewUser()
			if err := h.DB.DB(h.Database).C("users").FindId(admin).One(uadmin); err != nil {
				if err == mgo.ErrNotFound {
					return indexAlerts(c, http.StatusUnauthorized, "Invalid email", "danger")
				}
				c.Logger().Error(err)
				return indexAlerts(c, http.StatusUnauthorized, "Invalid email or password", "danger")
			}

			if !uadmin.IsAdmin() {
				return indexAlerts(c, http.StatusForbidden, "You are not an admin!", "danger")
			}

			if !u.ActivateAdmin(h.Key, hash, admin) {
				return indexAlerts(c, http.StatusBadRequest, "Invalid parameters", "danger")
			}

			if err := h.DB.DB(h.Database).C("users").UpdateId(email, bson.M{"$set": bson.M{"activated_by_admin": u.ActivatedByAdmin}}); err != nil {
				c.Logger().Error(err)
				return indexAlerts(c, http.StatusForbidden, "Invalid user email or hash", "danger")
			}

			if err := h.ES.Send([]string{u.Email}, "admin_registered", echo.Map{
				"url":     h.Url,
				"subject": "Account at CryProcessor web server is activated",
			}); err != nil {
				c.Logger().Error(err)
				return indexAlerts(c, http.StatusBadGateway, "Internal server error", "danger")
			}
		}
		return indexAlerts(c, http.StatusOK, "Account is activated", "success")
	}

	if !u.ActivateEmail(h.Key, hash) {
		return indexAlerts(c, http.StatusForbidden, "Invalid email or hash", "danger")
	}

	if err := h.DB.DB(h.Database).C("users").UpdateId(email, bson.M{"$set": bson.M{"activated_by_email": u.ActivatedByMail}}); err != nil {
		c.Logger().Error(err)
		return indexAlerts(c, http.StatusForbidden, "Invalid email or hash", "danger")
	}

	aiter := h.DB.DB(h.Database).C("users").Find(bson.M{"role": "admin"}).Iter()
	auser := models.NewUser()

	for aiter.Next(auser) {
		link, err := u.GetAdminActivationUrl(h.Key, h.Url, auser.Email)
		if err != nil {
			c.Logger().Error(err)
			return indexAlerts(c, http.StatusBadRequest, "Internal server error", "danger")
		}

		if err := h.ES.Send([]string{auser.Email}, "admin_registered", echo.Map{
			"link":    link,
			"subject": "Registration at CryProcessor web server",
			"user":    u,
		}); err != nil {
			c.Logger().Error(err)
			return indexAlerts(c, http.StatusBadGateway, "Internal server error", "danger")
		}
	}

	return indexAlerts(c, http.StatusOK, "Your email is confirmed. Your account is waiting for confirmation by admin. Your will be notified by email.", "success")
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

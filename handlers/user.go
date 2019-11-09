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
		return h.indexAlert(c, http.StatusBadRequest, "Malformed request", "danger")
	}

	if err := h.DB.DB(h.Database).
		C("users").FindId(u.Email).One(u); err != nil {
		if err == mgo.ErrNotFound {
			return h.indexAlert(c, http.StatusUnauthorized, "Invalid email or password", "danger")
		}
		c.Logger().Error(err)
		return h.indexAlert(c, http.StatusUnauthorized, "Invalid email or password", "danger")
	}

	pwd := c.FormValue("password")
	if !u.CheckPassword(pwd) {
		return h.indexAlert(c, http.StatusUnauthorized, "Invalid email or password", "danger")
	}

	if !u.IsActive() {
		return h.indexAlert(c, http.StatusForbidden, "Your account is waiting for activation", "danger")
	}

	expires := time.Hour * 24

	claims := &JwtUserClaims{
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
		Path:   "/",
	})

	if c.FormValue("redirect_url") != "" {
		return c.Redirect(http.StatusMovedPermanently, c.FormValue("redirect_url"))
	}

	return h.indexAlert(c, http.StatusOK, "You are logged in!", "success")
}

func (h *Handler) Register(c echo.Context) error {
	u := models.NewUser()

	if err := c.Bind(u); err != nil {
		c.Logger().Error(err)
		return h.indexAlert(c, http.StatusBadRequest, "Malformed request", "danger")
	}

	if err := c.Validate(u); err != nil {
		return h.indexAlert(c, http.StatusBadRequest, err.Error(), "danger")
	}

	pwd1, pwd2 := c.FormValue("password"), c.FormValue("password2")

	if err := validator.New().VarWithValue(pwd1, pwd2, "required,eqfield"); err != nil {
		return h.indexAlert(c, http.StatusBadRequest, "Passwords do not match", "danger")
	}

	if err := u.SetPassword(pwd1); err != nil {
		c.Logger().Error(err)
		return h.indexAlert(c, http.StatusBadRequest, "Failed to register new user", "danger")
	}

	if err := h.DB.DB(h.Database).C("users").Insert(u); err != nil {
		if mgo.IsDup(err) {
			return h.indexAlert(c, http.StatusForbidden, "User with the same email address already exists", "danger")
		}
		c.Logger().Error(err)
		return h.indexAlert(c, http.StatusBadRequest, "Failed to register new user", "danger")
	}

	link, err := u.GetActivationUrl(h.Key, h.Url, h.Route("user.activate"))
	if err != nil {
		c.Logger().Error(err)
		return h.indexAlert(c, http.StatusBadRequest, "Failed to register new user", "danger")
	}

	if err := h.ES.Send([]string{u.Email},
		"Registration at CryProcessor web server",
		"registered",
		echo.Map{
			"link": link,
		}); err != nil {
		if err2 := h.DB.DB(h.Database).C("users").RemoveId(u.Email); err2 != nil {
			c.Logger().Error(err2)
		}
		c.Logger().Error(err)
		return h.indexAlert(c, http.StatusBadGateway, "Failed to sent email with activation link", "danger")
	}

	return h.indexAlert(c, http.StatusOK, "You were successfully registered. Please, activate your account with the link in the email.", "success")
}

func (h *Handler) Activate(c echo.Context) error {
	email, hash, admin := c.FormValue("email"), c.FormValue("hash"), c.FormValue("admin")
	v := validator.New()

	if v.Var(email, "required,email") != nil || v.Var(hash, "required") != nil {
		return h.indexAlert(c, http.StatusBadGateway, "Invalid email or hash", "danger")
	}

	u := models.NewUser()

	if err := h.DB.DB(h.Database).
		C("users").FindId(email).One(u); err != nil {
		if err == mgo.ErrNotFound {
			return h.indexAlert(c, http.StatusUnauthorized, "Invalid email or password", "danger")
		}
		c.Logger().Error(err)
		return h.indexAlert(c, http.StatusUnauthorized, "Invalid email or password", "danger")
	}

	if v.Var(admin, "required,email") == nil {
		if !u.ActivatedByMail {
			return h.indexAlert(c, http.StatusForbidden, "User didn't confirm the email address", "danger")
		}
		if !u.IsActive() {
			uadmin := models.NewUser()
			if err := h.DB.DB(h.Database).C("users").FindId(admin).One(uadmin); err != nil {
				if err == mgo.ErrNotFound {
					return h.indexAlert(c, http.StatusUnauthorized, "Invalid email", "danger")
				}
				c.Logger().Error(err)
				return h.indexAlert(c, http.StatusUnauthorized, "Invalid email or password", "danger")
			}

			if !uadmin.IsAdmin() {
				return h.indexAlert(c, http.StatusForbidden, "You are not an admin!", "danger")
			}

			if !u.ActivateAdmin(h.Key, hash, admin) {
				return h.indexAlert(c, http.StatusBadRequest, "Invalid parameters", "danger")
			}

			if err := h.DB.DB(h.Database).C("users").UpdateId(email, u); err != nil {
				c.Logger().Error(err)
				return h.indexAlert(c, http.StatusForbidden, "Invalid user email or hash", "danger")
			}

			if err := h.ES.Send([]string{u.Email},
				"Account at CryProcessor web server is activated",
				"confirmed", echo.Map{
					"url": h.Url,
				}); err != nil {
				c.Logger().Error(err)
				return h.indexAlert(c, http.StatusBadGateway, "Internal server error", "danger")
			}
		} else {
			return h.indexAlert(c, http.StatusForbidden, "Invalid email or password", "danger")
		}
		return h.indexAlert(c, http.StatusOK, "Account is activated", "success")
	}

	if u.ActivatedByMail {
		return h.indexAlert(c, http.StatusForbidden, "Invalid email or password", "danger")
	}

	if !u.ActivateEmail(h.Key, hash) {
		c.Logger().Errorf("failed to activate: email <%s>, hash <%s>\n", email, hash)
		return h.indexAlert(c, http.StatusForbidden, "Invalid email or hash", "danger")
	}

	if err := h.DB.DB(h.Database).C("users").UpdateId(email, u); err != nil {
		c.Logger().Error(err)
		return h.indexAlert(c, http.StatusForbidden, "Invalid email or hash", "danger")
	}

	aiter := h.DB.DB(h.Database).C("users").Find(bson.M{"role": "admin"}).Iter()
	auser := models.NewUser()

	for aiter.Next(auser) {
		link, err := u.GetAdminActivationUrl(h.Key, h.Url, h.Route("user.activate"), auser.Email)
		if err != nil {
			c.Logger().Error(err)
			return h.indexAlert(c, http.StatusBadRequest, "Internal server error", "danger")
		}

		if err := h.ES.Send([]string{auser.Email},
			"Registration at CryProcessor web server",
			"admin_registered", echo.Map{
				"link": link,
				"user": u,
			}); err != nil {
			c.Logger().Error(err)
			return h.indexAlert(c, http.StatusBadGateway, "Internal server error", "danger")
		}
	}

	return h.indexAlert(c, http.StatusOK, "Your email is confirmed. Your account is waiting for confirmation by admin. Your will be notified by email.", "success")
}

func (h *Handler) LoginPage(ctx echo.Context) error {
	lg := ctx.Get("logged").(bool)
	if lg {
		userJwt := ctx.Get("user").(*jwt.Token)
		claims := userJwt.Claims.(*JwtUserClaims)
		user_id := claims.Email

		u := models.NewUser()

		if err := h.DbUser().FindId(user_id).One(u); err != nil {
			h.indexAlert(ctx, http.StatusOK, "Failed to get info about this user", "danger")
		}
		return ctx.Render(http.StatusForbidden, "pages/index", echo.Map{
			"tool_name":    "cry_processor", // TODO: Rewrite to normal tool name
			"logged":       true,
			"user":         u,
			"notification": "your are already logged in",
			"login_url":    h.Route("user.login"),
			"register_url": h.Route("user.register"),
		})
	}
	return ctx.Render(http.StatusOK, "pages/login", echo.Map{
		"redirect_url": ctx.QueryParam("redirect_url"),
		"register_url": h.Route("user.register"),
		"login_url":    h.Route("user.login"),
	})
}

func (h *Handler) RegisterPage(ctx echo.Context) error {
	u, l := h.checkLogged(ctx)
	if l {
		return ctx.Render(http.StatusForbidden, "pages/index", echo.Map{
			"tool_name":    "cry_processor", // TODO: ewrite to normal tool name
			"logged":       true,
			"user":         u,
			"notification": "your are already logged in",
			"login_url":    h.Route("user.login"),
			"register_url": h.Route("user.register"),
		})
	}
	return ctx.Render(http.StatusOK, "pages/register", echo.Map{
		"action_url": h.Route("user.register"),
		"cancel_url": h.Route("main.page"),
	})
}

type JwtUserClaims struct {
	Email string `json:"email"`
	jwt.StandardClaims
}

func (h *Handler) indexAlert(ctx echo.Context, code int, notification, alert string) error {
	u, l := h.checkLogged(ctx)
	return ctx.Render(code, "pages/index", echo.Map{
		"notification": notification,
		"alert_type":   alert,
		"logged":       l,
		"user":         u,
		"login_url":    h.Route("user.login"),
		"register_url": h.Route("user.register"),
	})
}

func (h *Handler) checkLogged(ctx echo.Context) (*models.User, bool) {
	lg := ctx.Get("logged").(bool)
	if lg {
		userJwt := ctx.Get("user").(*jwt.Token)
		claims := userJwt.Claims.(*JwtUserClaims)
		user_id := claims.Email

		u := models.NewUser()

		if err := h.DbUser().FindId(user_id).One(u); err != nil {
			return nil, false
		}
		return u, lg
	}
	return nil, lg
}

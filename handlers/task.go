package handlers

import (
	"io"
	"net/http"
	"os"
	"path"

	"github.com/dgrijalva/jwt-go"
	"github.com/lab7arriam/cryweb/models"
	"github.com/labstack/echo/v4"
	"gopkg.in/go-playground/validator.v9"
	"gopkg.in/mgo.v2/bson"
)

func (h *Handler) TasksList(c echo.Context) error {
	tool := c.Param("tool")
	userJwt := c.Get("user").(*jwt.Token)
	claims := userJwt.Claims.(*JwtUserClaims)
	user_id := claims.Email

	var tasks []*models.Task

	if err := h.DbTask().Find(bson.M{"tool": tool, "user_id": user_id}).Sort("-created").All(&tasks); err != nil {
		return h.indexAlert(c, http.StatusBadGateway, "failed to get list of tasks", "error")
	}

	return c.Render(http.StatusOK, "pages/tasks", echo.Map{
		"tasks":        tasks,
		"add_task_url": h.Route("tasks.add_form", tool),
	})
}

func (h *Handler) AddTask(c echo.Context) error {
	tool := c.Param("tool")
	userJwt := c.Get("user").(*jwt.Token)
	claims := userJwt.Claims.(*JwtUserClaims)
	user_id := claims.Email

	if count, err := h.DbTask().Find(bson.M{"tool": tool, "user_id": user_id, "status": "running"}).Count(); err != nil || count > 0 {
		if err != nil {
			c.Logger().Error(err)
			return h.indexAlert(c, http.StatusBadGateway, "failed to create task", "error")
		}
		if count > 0 {
			return h.indexAlert(c, http.StatusForbidden, "has running task", "error")
		}
	}

	vldtr := validator.New()

	//TODO: rewrite validation of tool from db
	if err := vldtr.VarWithValue(tool, "cry_processor", "required,eqfield"); err != nil {
		return h.indexAlert(c, http.StatusForbidden, "wrong tool name", "error")
	}

	mode := c.FormValue("run_mode")

	if err := vldtr.Var(mode, "required,oneof=proteins single meta"); err != nil {
		return h.indexAlert(c, http.StatusBadRequest, "malformed request", "error")
	}

	task, err := models.NewTask(user_id, tool, h.WorkDir)
	if err != nil {
		return h.indexAlert(c, http.StatusBadGateway, "failed to create task", "error")
	}

	task.AddParam("run_mode", mode)

	if mode == "proteins" {
		if err := saveFile(c, "protein_seq", "fi", task); err != nil {
			c.Logger().Error(err)
			return h.indexAlert(c, http.StatusBadRequest, "malformed request", "error")
		}
	} else {
		if err := saveFile(c, "forward_reads", "fo", task); err != nil {
			c.Logger().Error(err)
			return h.indexAlert(c, http.StatusBadRequest, "malformed request", "error")
		}
		if err := saveFile(c, "reverse_reads", "re", task); err != nil {
			c.Logger().Error(err)
			return h.indexAlert(c, http.StatusBadRequest, "malformed request", "error")
		}
		if mode == "meta" {
			task.AddParam("meta", "meta")
		}
	}

	if err := h.DbTask().Insert(task); err != nil {
		c.Logger().Error(err)
		return h.indexAlert(c, http.StatusBadGateway, "failed to create task", "error")
	}

	if _, err := h.Celery.Delay("go_cry", task.GetParam("run_mode"), task.GetParam("fi"),
		task.GetParam("fo"), task.GetParam("re"), task.GetParam("meta"),
		task.WorkDir); err != nil {
		c.Logger().Error(err)
		return h.indexAlert(c, http.StatusBadGateway, "failed to create task", "error")
	}

	return h.indexAlert(c, http.StatusOK, "task created", "success")
}

func (h *Handler) GetResults(c echo.Context) error {
	id := c.Param("task")
	userJwt := c.Get("user").(*jwt.Token)
	claims := userJwt.Claims.(*JwtUserClaims)
	user_id := claims.Email

	task := &models.Task{}

	if err := h.DbTask().FindId(bson.ObjectIdHex(id)).One(task); err != nil {
		c.Logger().Error(err)
		return h.indexAlert(c, http.StatusBadGateway, "failed to get task results", "error")
	}

	if !task.ResultAvailable(user_id) {
		return h.indexAlert(c, http.StatusForbidden, "failed to get task results", "error")
	}

	filename := path.Join(task.WorkDir, "cry_result.zip")

	return c.File(filename)
}

func saveFile(ctx echo.Context, name, param string, t *models.Task) error {
	file, err := ctx.FormFile(name)
	if err != nil {
		return err
	}
	src, err := file.Open()
	if err != nil {
		return err
	}
	defer src.Close()

	basename := path.Base(file.Filename)
	filename := path.Join(t.WorkDir, basename)

	dst, err := os.Create(filename)
	if err != nil {
		return err
	}

	if _, err = io.Copy(dst, src); err != nil {
		return err
	}

	t.AddParam(param, basename)

	return nil
}

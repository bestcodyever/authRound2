package actions

import (
	"fmt"

	"github.com/gobuffalo/buffalo"
	"github.com/gobuffalo/gothrecipe/models"
	"github.com/markbates/going/defaults"
	"github.com/markbates/goth"
	"github.com/markbates/goth/gothic"
	"github.com/markbates/goth/providers/github"
	"github.com/markbates/pop"
	"github.com/markbates/pop/nulls"
	"github.com/pkg/errors"
)

func init() {
	gothic.Store = App().SessionStore

	goth.UseProviders(
		// github.New(os.Getenv("GITHUB_KEY"), os.Getenv("GITHUB_SECRET"), "http://localhost:3000/auth/github/callback"),
		github.New("92839ddaf7129f39daaf", "97e718d3e94e8be831f9db54dfe9bb2820f8758c", "http://localhost:3000/auth/github/callback"),
	)
}

// AuthCallback auth func
func AuthCallback(c buffalo.Context) error {
	gu, err := gothic.CompleteUserAuth(c.Response(), c.Request())
	if err != nil {
		return c.Error(401, err)
	}
	tx := c.Value("tx").(*pop.Connection)
	q := tx.Where("provider = ? and provider_id = ?", gu.Provider, gu.UserID)
	exists, err := q.Exists("users")
	if err != nil {
		return errors.WithStack(err)
	}
	u := &models.User{}
	if exists {
		if err = q.First(u); err != nil {
			return errors.WithStack(err)
		}
	}
	u.Name = defaults.String(gu.Name, gu.NickName)
	u.Provider = gu.Provider
	u.ProviderID = gu.UserID
	u.Email = nulls.NewString(gu.Email)
	if err = tx.Save(u); err != nil {
		return errors.WithStack(err)
	}

	fmt.Println("set current_user_id")
	c.Session().Set("current_user_id", u.ID)
	if err = c.Session().Save(); err != nil {
		return errors.WithStack(err)
	}

	c.Flash().Add("success", "You have been logged in!")
	return c.Redirect(302, "/")
}

// AuthDestroy log out user
func AuthDestroy(c buffalo.Context) error {
	c.Session().Clear()
	err := c.Session().Save()
	if err != nil {
		return errors.WithStack(err)
	}
	c.Flash().Add("success", "You have been logged out!")
	return c.Redirect(302, "/")
}

// SetCurrentUser func
func SetCurrentUser(next buffalo.Handler) buffalo.Handler {
	return func(c buffalo.Context) error {
		if uid := c.Session().Get("current_user_id"); uid != nil {
			u := &models.User{}
			tx := c.Value("tx").(*pop.Connection)
			if err := tx.Find(u, uid); err != nil {
				return errors.WithStack(err)
			}
			c.Set("current_user", u)
		}
		return next(c)
	}
}

// Authorize func
func Authorize(next buffalo.Handler) buffalo.Handler {
	return func(c buffalo.Context) error {
		if uid := c.Session().Get("current_user_id"); uid == nil {
			c.Flash().Add("danger", "You must be authorized to see that page!")
			c.Redirect(302, "/")
		}
		return next(c)
	}
}

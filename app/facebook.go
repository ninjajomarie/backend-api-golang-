package external

import (
	fb "github.com/huandu/facebook"
)

type Facebook interface {
	GetSession(accessToken string) FacebookSession
}

type FacebookSession interface {
	Get(path string, params fb.Params) (fb.Result, error)
	Validate() error
}

type FacebookClient struct {
	FBApp *fb.App
}

func (c *FacebookClient) GetSession(accessToken string) FacebookSession {
	return c.FBApp.Session(accessToken)
}

package external_test

import (
	fb "github.com/huandu/facebook"
	external "github.com/johankaito/api.external/app"
)

type FakeFacebookClient struct {
	GetSessionLastCalledWith string
	GetSessionCallCount      int

	GetResponse       fb.Result
	GetLastCalledWith struct {
		Path   string
		Params fb.Params
	}
	GetCallCount      int
	GetError          error
	ValidateCallCount int
	ValidateError     error
}

func (c *FakeFacebookClient) GetSession(accessToken string) external.FacebookSession {
	c.GetSessionCallCount++
	c.GetSessionLastCalledWith = accessToken
	return &FakeFacebookSession{
		Client: c,
	}
}

type FakeFacebookSession struct {
	Client *FakeFacebookClient
}

func (f *FakeFacebookSession) Get(path string, params fb.Params) (fb.Result, error) {
	f.Client.GetCallCount++
	f.Client.GetLastCalledWith.Path = path
	f.Client.GetLastCalledWith.Params = params
	return f.Client.GetResponse, f.Client.GetError
}

func (f *FakeFacebookSession) Validate() error {
	f.Client.ValidateCallCount++
	return f.Client.ValidateError
}

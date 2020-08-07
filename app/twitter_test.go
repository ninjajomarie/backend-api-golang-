package external_test

import (
	external "github.com/johankaito/api.external/app"
)

type FakeTwitterClient struct {
	VerifyCredentialsResponse       *external.TwitterUser
	VerifyCredentialsLastCalledWith external.VerifyCredentialsParams
	VerifyCredentialsCallCount      int
	VerifyCredentialsError          error
}

func (c *FakeTwitterClient) VerifyCredentials(
	p external.VerifyCredentialsParams,
) (
	*external.TwitterUser, error,
) {
	c.VerifyCredentialsLastCalledWith = p
	c.VerifyCredentialsCallCount++
	return c.VerifyCredentialsResponse, c.VerifyCredentialsError
}

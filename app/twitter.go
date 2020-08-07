package external

import (
	"fmt"
	"net/http"
	"strings"

	tw "github.com/dghubble/go-twitter/twitter"
	"github.com/dghubble/oauth1"
	"github.com/pkg/errors"
)

type Twitter interface {
	VerifyCredentials(VerifyCredentialsParams) (*TwitterUser, error)
}

type TwitterClient struct {
	Config oauth1.Config
}

type TwitterUser struct {
	FirstName    string
	LastName     string
	EmailAddress string
}

type VerifyCredentialsParams struct {
	AccessToken  string
	AccessSecret string
}

func (c *TwitterClient) VerifyCredentials(p VerifyCredentialsParams) (
	*TwitterUser, error,
) {
	token := oauth1.NewToken(p.AccessToken, p.AccessSecret)
	httpClient := c.Config.Client(oauth1.NoContext, token)
	client := tw.NewClient(httpClient)
	includeEntities := true
	skipStatus := false
	includeEmail := true
	user, response, err := client.Accounts.VerifyCredentials(&tw.AccountVerifyParams{
		IncludeEntities: &includeEntities,
		SkipStatus:      &skipStatus,
		IncludeEmail:    &includeEmail,
	})
	if err != nil {
		return nil, errors.Wrap(err, "verifying credentials")
	}
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("non 200 code from twitter got %d", response.StatusCode)
	}

	nameArr := strings.Split(user.Name, " ")
	var lastName, firstName string
	l := len(nameArr)
	if l <= 1 {
		lastName = user.Name
	} else {
		firstName = strings.Join(nameArr[:l-1], " ")
		lastName = nameArr[l-1]
	}

	return &TwitterUser{
		EmailAddress: user.Email,
		FirstName:    firstName,
		LastName:     lastName,
	}, nil
}

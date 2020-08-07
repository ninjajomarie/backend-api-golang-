package external

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/pkg/errors"
)

var deviceIDHeader = "X-GGWP-Device-Unique-Id"

func (e *External) JWTAuthentication(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// list of endpoints that don't require auth
		notAuth := map[string][]string{
			http.MethodGet: []string{
				"/user/login",
			},
			http.MethodPost: []string{
				"/user",
			},
		}
		requestPath := r.URL.Path // current request path
		requestMethod := r.Method // current request path

		// check if request does not need authentication, serve the request if it doesn't need it
		for method, routes := range notAuth {
			if requestMethod == method {
				for _, route := range routes {
					if requestPath == route {
						next.ServeHTTP(w, r)
						return
					}
				}
			}
		}

		tokenHeader := r.Header.Get(AccessTokenHeader) // grab the token from the header

		if tokenHeader == "" { // token is missing, returns with error code 403 Unauthorized
			e.writeError(w, r, http.StatusForbidden, fmt.Errorf("missing auth token"))
			return
		}

		splitted := strings.Split(tokenHeader, " ") // the token normally comes in format `Bearer {token-body}`, we check if the retrieved token matched this requirement
		if len(splitted) != 2 {
			e.writeError(w, r, http.StatusForbidden, fmt.Errorf("invalid/malformed auth token"))
			return
		}

		tokenPart := splitted[1] // grab the token part, what we are truly interested in
		tk := &AccessToken{}

		token, err := jwt.ParseWithClaims(tokenPart, tk, func(token *jwt.Token) (interface{}, error) {
			return []byte(os.Getenv("TOKEN_PASSWORD")), nil
		})
		if err != nil { // malformed token, returns with http code 403 as usual
			e.writeError(w, r, http.StatusForbidden, fmt.Errorf("malformed auth token"))
			return
		}

		if !token.Valid { // token is invalid, maybe not signed on this server
			e.writeError(w, r, http.StatusForbidden, fmt.Errorf("token is not valid"))
			return
		}

		// everything went well, proceed with the request and set the caller to the user retrieved from the parsed token
		ctx := context.WithValue(r.Context(), "user_id", tk.UserID)
		r = r.WithContext(ctx)
		next.ServeHTTP(w, r) // proceed in the middleware chain!
	})
}

func (e *External) AdminAuthentication(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID := r.Context().Value("user_id").(int)
		if userID <= 0 {
			e.writeError(w, r, http.StatusForbidden, fmt.Errorf("missing user_id in context"))
			return
		}

		user, err := GetUserByID(e.dao.ReadDB, userID)
		if err != nil {
			e.writeError(w, r, http.StatusForbidden, errors.Wrapf(err, "unknown user"))
			return
		}

		if strings.ToLower(user.UserAdminLevel) != "admin" {
			e.writeError(w, r, http.StatusForbidden, fmt.Errorf("admin access required"))
			return
		}

		next.ServeHTTP(w, r) // proceed in the middleware chain!
	})
}

func (e *External) DeviceUniqueIDParser(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := context.WithValue(r.Context(), "device_unique_id", r.Header.Get(deviceIDHeader))
		r = r.WithContext(ctx)
		next.ServeHTTP(w, r) // proceed in the middleware chain!
	})
}

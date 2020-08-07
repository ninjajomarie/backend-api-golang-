package main

import (
	"net/http"
	"os"
	"time"

	"github.com/dghubble/oauth1"
	fb "github.com/huandu/facebook"
	external "github.com/johankaito/api.external/app"
	"github.com/sirupsen/logrus"
)

// Set by build
var version string

const (
	httpServerReadTimeout  = 60 * time.Second
	httpServerWriteTimeout = 120 * time.Second
)

func main() {
	logger := logrus.NewEntry(logrus.New()).WithField("version", "production")
	logger.Info("Starting server")
	cfg, err := getConfig()
	if err != nil {
		logger.WithError(err).Fatalf("reading config")
	}

	dao, err := external.NewDAO()
	if err != nil {
		logger.WithError(err).Fatal("getting dao object")
	}

	fb.RFC3339Timestamps = true
	facebook := &external.FacebookClient{
		FBApp: fb.New(cfg.fbAppID, cfg.fbAppSecret),
	}

	twitter := &external.TwitterClient{
		Config: oauth1.Config{
			ConsumerKey:    cfg.twitterConsumerKey,
			ConsumerSecret: cfg.twitterConsumerSecret,
		},
	}

	e := external.New(logger, dao, facebook, twitter)
	h, _, err := external.Router(e, logger, cfg.allowedOrigins)
	if err != nil {
		logger.WithError(err).Fatal("listening and serving")
	}

	var hasCertFile bool
	var hasKeyFile bool
	if _, err := os.Stat(cfg.tlsCertFile); err == nil {
		hasCertFile = true
	} else {
		logger.WithError(err).Warnf("tls missing cert file: %s", cfg.tlsCertFile)
	}
	if _, err := os.Stat(cfg.tlsKeyFile); err == nil {
		hasKeyFile = true
	} else {
		logger.WithError(err).Warnf("tls missing key file: %s", cfg.tlsKeyFile)
	}

	httpServer := &http.Server{
		Addr:         ":" + cfg.listenPort,
		ReadTimeout:  httpServerReadTimeout,
		WriteTimeout: httpServerWriteTimeout,
		Handler:      h,
	}

	logger.WithFields(logrus.Fields{
		"port": cfg.listenPort,
	}).Info("listening")

	// start crons
	e.RunCrons()

	// serve
	if hasKeyFile && hasCertFile {
		if err := httpServer.ListenAndServeTLS(cfg.tlsCertFile, cfg.tlsKeyFile); err != nil {
			logger.WithError(err).Fatal("listening and serving HTTPS")
		}
	} else {
		if err := httpServer.ListenAndServe(); err != nil {
			logger.WithError(err).Fatal("listening and serving HTTP")
		}
	}
}

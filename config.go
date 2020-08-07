package main

import "os"

type Config struct {
	listenPort            string
	tlsCertFile           string
	tlsKeyFile            string
	fbAppID               string
	fbAppSecret           string
	twitterConsumerKey    string
	twitterConsumerSecret string
	twitterTokenURL       string
	allowedOrigins        []string
}

func getConfig() (*Config, error) {
	return &Config{
		listenPort:            os.Getenv("LISTEN_PORT"),
		tlsCertFile:           os.Getenv("TLS_CERT"),
		tlsKeyFile:            os.Getenv("TLS_KEY"),
		fbAppID:               os.Getenv("FB_APP_ID"),
		fbAppSecret:           os.Getenv("FB_APP_SECRET"),
		twitterConsumerKey:    os.Getenv("TWITTER_CONSUMER_KEY"),
		twitterConsumerSecret: os.Getenv("TWITTER_CONSUMER_SECRET"),
		twitterTokenURL:       os.Getenv("TWITTER_TOKEN_URL"),
	}, nil
}

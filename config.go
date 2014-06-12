package main

import (
	"log"
	"strings"
	"time"

	"github.com/spf13/pflag"
)

type Config struct {
	// Forerunner
	ShutdownTimeout time.Duration
	LogLevel        int

	// Docker
	Image          string
	Registry       string
	DockerEndpoint string
	Name           string

	// Etcd
	EtcdHosts    string
	ConfigPrefix string
}

func NewConfig() *Config {
	c := new(Config)

	pflag.StringVar(&c.Image, "image", "", "docker image to run")
	pflag.StringVar(&c.Registry, "registry", "", "docker registry to contact")
	pflag.StringVar(&c.Name, "name", "", "name of docker container")
	pflag.StringVar(&c.DockerEndpoint, "docker-endpoint", "unix:///var/run/docker.sock", "docker socket to use")
	pflag.DurationVar(&c.ShutdownTimeout, "shutdown-timeout", 5*time.Second, "how long to wait after intterupt before forcibly stopping")
	pflag.StringVar(&c.EtcdHosts, "etcd-hosts", "http://127.0.0.1:4001", "comma-separated list of etcd hosts to connect to")
	pflag.StringVar(&c.ConfigPrefix, "config-prefix", "/forerunner/", "etcd prefix to pull configuration from")

	logLevel := pflag.String("log-level", "info", "level to log at (debug, info, warning, error, fatal)")

	pflag.Parse()

	switch strings.ToLower(*logLevel) {
	case "debug":
		c.LogLevel = LevelDebug
	case "info":
		c.LogLevel = LevelInfo
	case "warning":
		c.LogLevel = LevelWarning
	case "error":
		c.LogLevel = LevelError
	case "fatal":
		c.LogLevel = LevelFatal
	default:
		log.Fatalf("%s is not a valid log level", logLevel)
	}

	return c
}

package main

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/spf13/pflag"
)

type Config struct {
	// Forerunner
	ShutdownTimeout time.Duration
	LogLevel        int

	Group string
	ID    string

	// Docker
	Image          string
	Registry       string
	DockerEndpoint string

	// Etcd
	EtcdHosts    string
	ConfigPrefix string
}

func NewConfig() *Config {
	c := new(Config)

	set := pflag.NewFlagSet("forerunner", pflag.ExitOnError)

	set.DurationVar(&c.ShutdownTimeout, "shutdown-timeout", 5*time.Second, "how long to wait after interupt before forcibly stopping")
	set.StringVar(&c.Group, "group", "", "this service's group")
	set.StringVar(&c.ID, "id", "", "this service's ID")

	logLevel := set.String("log-level", "info", "level to log at (debug, info, warning, error, fatal)")

	set.StringVar(&c.Image, "image", "", "docker image to run")
	set.StringVar(&c.Registry, "registry", "", "docker registry to contact")
	set.StringVar(&c.DockerEndpoint, "docker-endpoint", "unix:///var/run/docker.sock", "docker socket to use")

	set.StringVar(&c.EtcdHosts, "etcd-hosts", "http://127.0.0.1:4001", "comma-separated list of etcd hosts to connect to")
	set.StringVar(&c.ConfigPrefix, "config-prefix", "/forerunner/", "etcd prefix to pull configuration from")

	err := set.Parse(os.Args)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

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

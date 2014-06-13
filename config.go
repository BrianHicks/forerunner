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
	Command        []string
	DNS            string
	Registry       string
	DockerEndpoint string

	// Etcd
	EtcdHosts    string
	ConfigPrefix string

	// Health
	TCPHealthPort int64
	TCPHealthHost string
}

func NewConfig() *Config {
	c := new(Config)

	set := pflag.NewFlagSet("forerunner", pflag.ExitOnError)

	set.DurationVar(&c.ShutdownTimeout, "shutdown-timeout", 5*time.Second, "how long to wait after interrupt before forcibly stopping")
	set.StringVar(&c.Group, "group", "", "this service's group")
	set.StringVar(&c.ID, "id", "", "this service's ID")

	logLevel := set.String("log-level", "info", "level to log at (debug, info, warning, error, fatal)")

	set.StringVar(&c.Image, "image", "", "docker image to run")
	set.StringVar(&c.DNS, "dns", "", "DNS host to use for container")
	set.StringVar(&c.Registry, "registry", "", "docker registry to contact")
	set.StringVar(&c.DockerEndpoint, "docker-endpoint", "unix:///var/run/docker.sock", "docker socket to use")

	set.StringVar(&c.EtcdHosts, "etcd-hosts", "http://127.0.0.1:4001", "comma-separated list of etcd hosts to connect to")
	set.StringVar(&c.ConfigPrefix, "config-prefix", "/forerunner/", "etcd prefix to pull configuration from")

	set.Int64Var(&c.TCPHealthPort, "tcp-health-port", 0, "container port to check over TCP")
	set.StringVar(&c.TCPHealthHost, "tcp-health-host", "127.0.0.1", "container host")

	err := set.Parse(os.Args)

	c.Command = set.Args()[1:] // coalesce the rest of the args into arguments to the docker container

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

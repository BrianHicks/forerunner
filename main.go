package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"time"

	"github.com/fsouza/go-dockerclient"
)

var (
	router       *Router
	config       *Config
	dockerClient *DockerWrapper

	TopicInit        = "init"
	TopicShutdown    = "shutdown"
	TopicEnvironment = "environment"
	TopicDocker      = "docker"
	TopicTCPHealth   = "tcp-health"
)

func init() {
	router = NewRouter()
	config = NewConfig()

	client, err := docker.NewClient(config.DockerEndpoint)
	if err != nil {
		log.Fatal(err)
	}

	dockerClient = &DockerWrapper{client, docker.AuthConfiguration{}}
}

func main() {
	router.Register(EnvironmentListener, TopicInit, TopicShutdown)
	router.Register(DockerListener, TopicInit, TopicShutdown, TopicEnvironment)
	router.Register(TCPHealthListener, TopicInit, TopicShutdown, TopicDocker)

	router.Register(LogListener, TopicInit, TopicShutdown, TopicDocker, TopicEnvironment, TopicTCPHealth)

	router.In <- Message{
		Topic: TopicInit,
		Level: LevelInfo,
		Sent:  time.Now(),
	}

	// listen to OS signals to send the shutdown signal when appropriate
	stop := make(chan os.Signal)
	kill := make(chan os.Signal)

	signal.Notify(stop, os.Interrupt)
	signal.Notify(kill, os.Kill)

	for {
		select {
		case _ = <-stop:
			router.In <- Message{
				Topic:   TopicShutdown,
				Level:   LevelFatal,
				Message: fmt.Sprintf("interrupted, waiting %s to finish", config.ShutdownTimeout),
				Sent:    time.Now(),
			}
			time.Sleep(config.ShutdownTimeout)
			os.Exit(0)

		case _ = <-kill:
			fmt.Println(Message{
				Topic:   TopicShutdown,
				Level:   LevelFatal,
				Message: "kill signal received, halting immediately",
				Sent:    time.Now(),
			})
			os.Exit(0)
		}
	}
}

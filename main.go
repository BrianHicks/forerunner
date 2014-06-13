package main

import (
	"fmt"
	"os"
	"os/signal"
	"time"
)

var (
	router *Router
	config *Config

	TopicInit        = "init"
	TopicShutdown    = "shutdown"
	TopicEnvironment = "environment"
	TopicDocker      = "docker"
)

func init() {
	router = NewRouter()
	config = NewConfig()
}

func main() {
	router.Register(DockerListener, TopicInit, TopicShutdown, TopicEnvironment)
	router.Register(LogListener, TopicInit, TopicShutdown, TopicDocker, TopicEnvironment)
	router.Register(EnvironmentListener, TopicInit, TopicShutdown)

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

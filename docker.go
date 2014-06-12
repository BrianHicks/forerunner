package main

import (
	"log"
	"os"

	"github.com/fsouza/go-dockerclient"
)

func DockerListener(in chan Message, out chan Message) {
	send := Messenger(TopicDocker, out)

	client, err := docker.NewClient(config.DockerEndpoint)
	if err != nil {
		out <- MessageFromError(TopicDocker, LevelFatal, err)
		return
	}

	auth := docker.AuthConfiguration{}

	// listen for messages
	for message := range in {
		switch message.Topic {
		case TopicInit:
			// pull image
			if config.Image == "" {
				log.Fatal("Image is required for Docker")
			}

			send(LevelInfo, "pulling "+config.Image+":"+config.Tag)
			err := client.PullImage(
				docker.PullImageOptions{
					Repository:   config.Image,
					Registry:     config.Registry,
					Tag:          config.Tag,
					OutputStream: os.Stderr,
				},
				auth,
			)
			if err != nil {
				out <- MessageFromError(TopicDocker, LevelFatal, err)
				return
			}

		case TopicConfigChange:
			send(LevelDebug, "configuration changed, (re)starting docker image")

		case TopicShutdown:
			send(LevelDebug, "not running, nothing to shut down")
			return

		default:
			log.Fatalf("Docker can't process %s messages", message.Topic)
		}
	}
}

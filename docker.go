package main

import (
	"errors"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/fsouza/go-dockerclient"
)

var (
	ErrNoSuchContainer = errors.New("no such container")
	ErrNoPublicport    = errors.New("no public port")
)

type DockerWrapper struct {
	*docker.Client
	Auth docker.AuthConfiguration
}

func (d *DockerWrapper) Pull(image, tag, repository string) error {
	return d.PullImage(
		docker.PullImageOptions{
			Repository: image,
			Tag:        tag,
			Registry:   repository,
		},
		d.Auth,
	)
}

func (d *DockerWrapper) ContainerByName(name string) (*docker.APIContainers, error) {
	name = "/" + name
	containers, err := d.ListContainers(docker.ListContainersOptions{All: true})

	if err != nil {
		return nil, err
	}

	for _, container := range containers {
		for _, cname := range container.Names {
			if cname == name {
				return &container, nil
			}
		}
	}

	return nil, ErrNoSuchContainer
}

func (d *DockerWrapper) CompletelyKill(id string) error {
	err := d.KillContainer(docker.KillContainerOptions{ID: id})
	if err != nil {
		return err
	}

	err = d.RemoveContainer(docker.RemoveContainerOptions{ID: id})
	if err != nil {
		return err
	}

	return nil
}

func (d *DockerWrapper) CreateAndStart(name string, config *docker.Config, host *docker.HostConfig) (*docker.Container, error) {
	container, err := d.CreateContainer(docker.CreateContainerOptions{
		Name:   name,
		Config: config,
	})
	if err != nil {
		return nil, err
	}

	err = d.StartContainer(container.ID, host)

	return container, err
}

func (d *DockerWrapper) PublicPort(name string, private int64) (int64, error) {
	var port int64 = 0

	container, err := d.ContainerByName(name)
	if err != nil {
		return port, err
	}

	for _, maybe := range container.Ports {
		if maybe.Type != "tcp" {
			continue
		}

		if maybe.PrivatePort == private {
			port = maybe.PublicPort
		}
	}

	if port == 0 {
		return port, ErrNoPublicport
	} else {
		return port, nil
	}
}

func DockerListener(in, out chan Message) {
	send := Messenger(TopicDocker, out)
	name := config.Group + "-" + config.ID

	var timer *time.Timer
	timeout := 2 * time.Second
	lock := sync.Mutex{}

	// listen for messages
	for message := range in {
		switch message.Topic {
		case TopicInit:
			// pull image
			if config.Image == "" {
				log.Fatal("Image is required for Docker")
			}
			send(LevelInfo, fmt.Sprintf("docker ready (%s)", name))

		case TopicEnvironment:
			// reset here because environment changes can come together pretty
			// quickly, and we only want to restart once per group of changes.
			if message.Level < LevelChange {
				continue
			}

			reset := func() {
				lock.Lock()
				defer lock.Unlock()

				//// PULL IMAGE ////
				img := config.Image + ":" + EtcdTag
				send(LevelInfo, "pulling "+img)

				err := dockerClient.Pull(config.Image, EtcdTag, config.Registry)
				if err != nil {
					send(
						LevelError,
						fmt.Sprintf("error pulling %s: %s", img, err),
					)
					return
				}

				send(LevelDebug, "pulled "+img)

				//// START ////
				container, err := dockerClient.ContainerByName(name)

				// first, clean up old containers. We don't know what
				// configuration they're running so we're just going to restart
				// the container with the new configuration.
				if err != nil && err != ErrNoSuchContainer {
					send(LevelFatal, fmt.Sprintf("error getting containers: %s", err))
					return

				} else if err == ErrNoSuchContainer {
					send(LevelDebug, "no container running")

				} else {
					send(LevelInfo, "container running, cleaning before restart")

					err = dockerClient.CompletelyKill(container.ID)
					if err != nil {
						if strings.HasPrefix(err.Error(), "No such container") {
							send(LevelWarning, fmt.Sprintf("%s", err))
						} else {
							send(LevelFatal, fmt.Sprintf("%s", err))
							return
						}
					}
				}

				// now we start the container with the current configuration
				send(LevelDebug, "starting new container")
				_, err = dockerClient.CreateAndStart(
					name,
					&docker.Config{
						Image: img,
						// TODO: command
						// TODO: env
					},
					&docker.HostConfig{
						PublishAllPorts: true,
						// TODO: dns
						// TODO: ports
					},
				)

				if err != nil {
					send(LevelFatal, fmt.Sprintf("could not start container: %s", err))
				}
				send(LevelChange, "container running")

				timer = nil
			}

			if timer == nil {
				send(LevelInfo, fmt.Sprintf("detected configuration change, waiting for %s to reset", timeout))
				timer = time.AfterFunc(timeout, reset)
			} else if timer.Reset(timeout) {
				send(LevelDebug, fmt.Sprintf("additional configuration changes, resetting timer to %s", timeout))
			} else {
				timer = time.AfterFunc(timeout, reset)
			}

		case TopicShutdown:
			container, err := dockerClient.ContainerByName(name)

			if err != nil && err != ErrNoSuchContainer {
				send(LevelFatal, fmt.Sprintf("error getting containers: %s", err))

			} else if err == ErrNoSuchContainer {
				send(LevelDebug, "no container running")

			} else {
				send(LevelInfo, "shutting down container")

				err = dockerClient.CompletelyKill(container.ID)
				if err != nil {
					if strings.HasPrefix(err.Error(), "No such container") {
						send(LevelWarning, fmt.Sprintf("%s", err))
					} else {
						send(LevelFatal, fmt.Sprintf("%s", err))
					}
				}
			}
			return

		default:
			log.Fatalf("Docker can't process %s messages", message.Topic)
		}
	}
}

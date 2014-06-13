package main

import (
	"errors"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/coreos/go-etcd/etcd"
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

	etcdClient := etcd.NewClient(strings.Split(config.EtcdHosts, ","))

	// listen for messages
	for message := range in {
		switch message.Topic {
		case TopicInit:
			// pull image
			if config.Image == "" {
				log.Fatal("Image is required for Docker")
			}
			send(LevelInfo, StatusGood, fmt.Sprintf("docker ready (%s)", name))

		case TopicEnvironment:
			// reset here because environment changes can come together pretty
			// quickly, and we only want to restart once per group of changes.
			if message.Level < LevelChange {
				continue
			}

			reset := func() {
				lock.Lock()
				defer lock.Unlock()
				defer func() {
					timer = nil
				}()

				//// PULL IMAGE ////
				img := config.Image + ":" + EtcdTag
				send(LevelInfo, StatusNeutral, "pulling "+img)

				err := dockerClient.Pull(config.Image, EtcdTag, config.Registry)
				if err != nil {
					send(
						LevelError, StatusBad,
						fmt.Sprintf("error pulling %s: %s", img, err),
					)
					return
				}

				send(LevelDebug, StatusNeutral, "pulled "+img)

				// acquire lock
				lockPath := "/forerunner/locks/" + config.Group
				send(LevelDebug, StatusNeutral, fmt.Sprintf("trying to acquire lock at %s", lockPath))

				// we're going to give it 100 tries (60 seconds) to acquire a
				// lock for restarting, but if we can't get it after that long
				// we'll assume a deadlock and just proceed without one
				for i := 0; i < 100; i++ {
					_, err := etcdClient.Create(lockPath, "", 10)

					if err != nil {
						if strings.Contains(err.Error(), "Key already exists") {
							time.Sleep(600 * time.Millisecond)
							continue
						} else {
							send(LevelFatal, StatusBad, err.Error())
							return
						}

					} else {
						send(LevelDebug, StatusNeutral, fmt.Sprintf("got lock after %d tries", i))

						// defer releasing that lock after we're done
						defer etcdClient.Delete(lockPath, false)

						break
					}
				}

				//// START ////
				container, err := dockerClient.ContainerByName(name)

				// first, clean up old containers. We don't know what
				// configuration they're running so we're just going to restart
				// the container with the new configuration.
				if err != nil && err != ErrNoSuchContainer {
					send(LevelFatal, StatusBad, fmt.Sprintf("error getting containers: %s", err))
					return

				} else if err == ErrNoSuchContainer {
					send(LevelDebug, StatusNeutral, "no container running")

				} else {
					send(LevelInfo, StatusNeutral, "container running, cleaning before restart")

					err = dockerClient.CompletelyKill(container.ID)
					if err != nil {
						if strings.HasPrefix(err.Error(), "No such container") {
							send(LevelWarning, StatusBad, fmt.Sprintf("%s", err))
						} else {
							send(LevelFatal, StatusBad, fmt.Sprintf("%s", err))
							return
						}
					}
				}

				// now we start the container with the current configuration
				send(LevelDebug, StatusNeutral, "starting new container")

				conf := docker.Config{
					Image: img,
					Env:   EtcdEnv,
					Cmd:   config.Command,
				}

				host := docker.HostConfig{
					PublishAllPorts: true,
				}
				if config.DNS != "" {
					host.Dns = strings.Split(config.DNS, ",")
				}

				_, err = dockerClient.CreateAndStart(name, &conf, &host)

				if err != nil {
					send(LevelFatal, StatusBad, fmt.Sprintf("could not start container: %s", err))
				} else {
					send(LevelChange, StatusUp, "container running")
				}
			}

			if timer == nil {
				send(LevelInfo, StatusNeutral, fmt.Sprintf("detected configuration change, waiting for %s to reset", timeout))
				timer = time.AfterFunc(timeout, reset)
			} else if timer.Reset(timeout) {
				send(LevelDebug, StatusNeutral, fmt.Sprintf("additional configuration changes, resetting timer to %s", timeout))
			} else {
				timer = time.AfterFunc(timeout, reset)
			}

		case TopicShutdown:
			container, err := dockerClient.ContainerByName(name)

			if err != nil && err != ErrNoSuchContainer {
				send(LevelFatal, StatusBad, fmt.Sprintf("error getting containers: %s", err))

			} else if err == ErrNoSuchContainer {
				send(LevelDebug, StatusNeutral, "no container running")

			} else {
				send(LevelInfo, StatusDown, "shutting down container")

				err = dockerClient.CompletelyKill(container.ID)
				if err != nil {
					if strings.HasPrefix(err.Error(), "No such container") {
						send(LevelWarning, StatusNeutral, fmt.Sprintf("%s", err))
					} else {
						send(LevelFatal, StatusNeutral, fmt.Sprintf("%s", err))
					}
				}
			}
			return

		default:
			log.Fatalf("Docker can't process %s messages", message.Topic)
		}
	}
}

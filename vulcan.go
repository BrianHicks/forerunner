package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/coreos/fleet/etcd"
)

func VulcanListener(in, out chan Message) {
	send := Messenger(TopicVulcan, out)

	etcdClient := etcd.NewClient(strings.Split(config.EtcdHosts, ","))

	listenTo := TopicDocker
	key := ""
	registered := false
	var (
		timeout uint64 = 10
		tick    <-chan time.Time
	)

	register := func() (success bool) {
		ip, err := dockerClient.PublicPort(config.Group+"-"+config.ID, config.PublicPort)
		if err != nil {
			send(LevelError, StatusBad, err.Error())
			return false
		}

		_, err = etcdClient.Set(key, fmt.Sprintf("%s:%d", config.PublicHost, ip), timeout)
		if err != nil {
			send(LevelError, StatusBad, err.Error())
			return false
		}

		return true
	}

	deregister := func() (success bool) {
		_, err := etcdClient.Delete(key, false)
		if err != nil {
			send(LevelError, StatusBad, err.Error())
			return false
		}

		return true
	}

	for {
		select {
		case message := <-in:
			switch message.Topic {
			case TopicInit:
				if !config.RegisterVulcan {
					return
				}

				if config.PublicPort == 0 {
					send(LevelError, StatusBad, "cannot register a service on port 0")
					return
				}

				if config.TCPHealthPort != 0 {
					listenTo = TopicTCPHealth
				}

				send(LevelDebug, StatusNeutral, "setting up vulcan")
				key = fmt.Sprintf("/vulcand/upstreams/%s/endpoints/%s", config.Group, config.ID)
				tick = time.Tick(time.Duration(timeout/2) * time.Second)

			case TopicShutdown:
				deregister()
				send(LevelChange, StatusDown, "tearing down vulcan")

			case listenTo:
				if message.Level != LevelChange {
					continue
				}

				switch message.Status {
				case StatusUp:
					if !registered {
						success := register()
						if success {
							registered = true
							send(LevelChange, StatusUp, fmt.Sprintf("registered %s", key))
						}
					}

				case StatusDown:
					if registered {
						success := deregister()
						if success {
							registered = false
							send(LevelChange, StatusDown, fmt.Sprintf("deregistered %s", key))
						}
					}
				}
			}

		case <-tick:
			if registered {
				register()
			}
		}
	}
}

package main

import (
	"fmt"
	"net"
	"time"
)

func TCPHealthListener(in, out chan Message) {
	send := Messenger(TopicTCPHealth, out)

	var (
		name string
		tick <-chan time.Time
	)

	started := false

	health := NewHealth(5)
	checks := make(chan bool, 1)
	status := health.Watch(checks)

	for {
		select {
		case message := <-in:
			switch message.Topic {
			case TopicInit:
				if config.TCPHealthPort == 0 {
					send(LevelDebug, "no port set, TCP health exiting")
				}
				name = config.Group + "-" + config.ID

			case TopicDocker:
				// start pinging!
				if started || message.Level != LevelChange {
					continue
				}

				send(LevelInfo, fmt.Sprintf("healthcheck starting on %s", config.TCPHealthHost))
				tick = time.Tick(5 * time.Second)
				started = true
			}

		case <-tick:
			port, err := dockerClient.PublicPort(name, config.TCPHealthPort)
			if err != nil {
				send(LevelDebug, err.Error())
				checks <- false
				continue
			}

			_, err = net.Dial("tcp", fmt.Sprintf("%s:%d", config.TCPHealthHost, port))
			if err != nil {
				send(LevelDebug, err.Error())
				checks <- false
				continue
			}

			send(LevelDebug, "check passed")
			checks <- true

		case current := <-status:
			send(LevelChange, fmt.Sprintf("healthy: %t", current))
		}
	}
}

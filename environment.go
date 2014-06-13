package main

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/coreos/go-etcd/etcd"
)

var (
	EtcdEnvironment = map[string]string{}
	EtcdTag         = "latest"
)

func EnvironmentListener(in, out chan Message) {
	send := Messenger(TopicEnvironment, out)

	client := etcd.NewClient(strings.Split(config.EtcdHosts, ","))

	tagKey := config.ConfigPrefix + "tag"
	envKey := config.ConfigPrefix + "env"

	watch := make(chan *etcd.Response, 10)
	watchStop := make(chan bool)

	for {
		select {
		case message := <-in:
			switch message.Topic {
			case TopicInit:
				send(LevelInfo, fmt.Sprintf("setting watch on %s", config.ConfigPrefix))
				go client.Watch(config.ConfigPrefix, 0, true, watch, watchStop)

				send(LevelInfo, "getting initial configuration from etcd")
				resp, err := client.Get(tagKey, false, false)
				if err != nil {
					send(LevelFatal, fmt.Sprintf("failed to get tag: %s", err))
				}
				watch <- resp

				resp, err = client.Get(envKey, false, false)
				if err != nil {
					send(LevelFatal, fmt.Sprintf("failed to get env: %s", err))
				}
				watch <- resp

			case TopicShutdown:
				watchStop <- true
				send(LevelInfo, fmt.Sprintf("cleared watches on %s", config.ConfigPrefix))
				return
			}

		case resp := <-watch:
			if resp == nil {
				send(LevelWarning, "received a nil response")
				continue
			}

			switch resp.Node.Key {
			case tagKey:
				EtcdTag = resp.Node.Value
				send(LevelChange, fmt.Sprintf("tag is %s", EtcdTag))

			case envKey:
				err := json.Unmarshal([]byte(resp.Node.Value), &EtcdEnvironment)
				if err != nil {
					send(LevelFatal, fmt.Sprintf("error loading env: %s", err))
				}
				send(LevelChange, fmt.Sprintf("environment is %s", EtcdEnvironment))

			default:
				send(LevelDebug, fmt.Sprintf("unknown config key: %s", resp.Node.Key))
			}
		}
	}
}

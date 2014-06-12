package main

import (
	"fmt"
	"os"
)

func LogListener(in, out chan Message) {
	for message := range in {
		if message.Level >= config.LogLevel {
			fmt.Println(message)
		}
		// exit immediately for fatal messages, except if they're for shutdown
		// (where `main` takes care of the timeout)
		if message.Level == LevelFatal && message.Topic != TopicShutdown {
			os.Exit(1)
		}
	}
}

package main

import (
	"fmt"
	"sync"
	"time"
)

const (
	LevelDebug = iota
	LevelInfo
	LevelWarning
	LevelError
	LevelFatal
)

var names = []string{"debug", "info", "warning", "error", "fatal"}

type Message struct {
	Topic   string
	Level   int
	Message string
	Sent    time.Time
}

func (m Message) String() string {
	s := fmt.Sprintf(
		"[%s] %s (%s)",
		m.Sent.Format(time.RFC3339),
		m.Topic,
		names[m.Level],
	)
	if m.Message != "" {
		s += ": " + m.Message
	}
	return s
}

type Router struct {
	lock      sync.RWMutex
	receivers map[string][]chan Message
	In        chan Message
}

func NewRouter() *Router {
	r := Router{
		sync.RWMutex{},
		map[string][]chan Message{},
		make(chan Message),
	}

	go r.Route()

	return &r
}

func (r *Router) Register(subscriber func(chan Message, chan Message), topics ...string) {
	sub := make(chan Message)
	r.Subscribe(sub, topics...)

	go subscriber(sub, r.In)
}

func (r *Router) Subscribe(subscriber chan Message, topics ...string) {
	r.lock.Lock()
	defer r.lock.Unlock()

	for _, topic := range topics {
		rec, ok := r.receivers[topic]
		if !ok {
			r.receivers[topic] = []chan Message{subscriber}
		} else {
			r.receivers[topic] = append(rec, subscriber)
		}
	}
}

func (r *Router) Route() {
	for message := range r.In {
		r.lock.RLock()
		recs, ok := r.receivers[message.Topic]
		r.lock.RUnlock()

		if !ok {
			continue
		}

		for _, rec := range recs {
			rec <- message
		}
	}
}

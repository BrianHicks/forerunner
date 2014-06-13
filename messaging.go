package main

import (
	"fmt"
	"sync"
	"time"
)

const (
	LevelDebug = iota
	LevelInfo
	LevelChange
	LevelWarning
	LevelError
	LevelFatal
)

type Status string

var (
	StatusNeutral Status = "neutral"
	StatusGood    Status = "good"
	StatusBad     Status = "bad"
	StatusUp      Status = "up"
	StatusDown    Status = "down"
)

var names = []string{"debug", "info", "change", "warning", "error", "fatal"}

type Message struct {
	Topic   string
	Level   int
	Status  Status
	Message string
	Sent    time.Time
}

func MessageFromError(topic string, level int, err error) Message {
	return Message{
		Topic:   topic,
		Level:   level,
		Message: fmt.Sprintf("%s", err),
		Sent:    time.Now(),
	}
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
		make(chan Message, 10),
	}

	go r.Route()

	return &r
}

func (r *Router) Register(subscriber func(chan Message, chan Message), topics ...string) {
	sub := make(chan Message, 10)
	r.Subscribe(sub, topics...)

	go subscriber(sub, r.In)
}

func (r *Router) Subscribe(subscriber chan Message, topics ...string) {
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
		recs, ok := r.receivers[message.Topic]

		if !ok {
			continue
		}

		for _, rec := range recs {
			rec <- message
		}
	}
}

func Messenger(topic string, to chan Message) func(int, Status, string) {
	return func(level int, status Status, message string) {
		to <- Message{
			Topic:   topic,
			Level:   level,
			Status:  status,
			Message: message,
			Sent:    time.Now(),
		}
	}
}

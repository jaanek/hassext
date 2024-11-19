package hub

import (
	"context"
	"fmt"

	"github.com/zerodha/logf"
)

type Client interface {
	Receive([]byte) error
}

type Registration struct {
	Topic  string
	Client Client
}

type Message struct {
	Topic string
	Data  []byte
}

type Hub struct {
	lo         logf.Logger
	clients    map[string][]Client
	Broadcast  chan Message
	Register   chan Registration
	Unregister chan Registration
}

func New(lo logf.Logger) *Hub {
	return &Hub{
		lo:         lo,
		clients:    make(map[string][]Client),
		Broadcast:  make(chan Message),
		Register:   make(chan Registration),
		Unregister: make(chan Registration),
	}
}

func (h *Hub) Run(ctx context.Context) {
	for {
		select {
		case r := <-h.Register:
			h.clients[r.Topic] = append(h.clients[r.Topic], r.Client)
			h.lo.Info(fmt.Sprintf("New client registered for topic: %s", r.Topic))
		case r := <-h.Unregister:
			clients := h.clients[r.Topic]
			if len(clients) <= 0 {
				// no clients registered
				return
			}
			// remove client
			nclients := make([]Client, len(clients)-1)
			for i, n := 0, 0; i < len(clients); i++ {
				c := clients[i]
				if c == r.Client {
					continue
				}
				nclients[n] = c
				n++
			}
			h.clients[r.Topic] = nclients
			h.lo.Info(fmt.Sprintf("Client unregistered from topic: %s", r.Topic))
		case m := <-h.Broadcast:
			clients := h.clients[m.Topic]
			h.lo.Info(fmt.Sprintf("Message received from topic: %s. Delivering to registered clients (%v). Message: %s", m.Topic, len(clients), string(m.Data)))
			for _, c := range clients {
				err := c.Receive(m.Data)
				if err != nil {
					h.lo.Warn(fmt.Sprintf("Message not delivered for client! Topic: %s", m.Topic))
				}
			}
		case <-ctx.Done():
			return
		}
	}
}

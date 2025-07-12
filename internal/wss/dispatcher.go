package wss

import (
	"errors"
	"log"

	wsstypes "github.com/lijuuu/ChallengeWssManagerService/internal/wss/types"
)

// WsHandlerType defines the signature for a WebSocket event handler
type WsHandlerType func(*wsstypes.WsContext) error

type Dispatcher struct {
	handlers map[string]WsHandlerType
}

func NewDispatcher() *Dispatcher {
	log.Println("dispatcher initialized")
	return &Dispatcher{
		handlers: make(map[string]WsHandlerType),
	}
}

func (d *Dispatcher) Register(event string, handler WsHandlerType) {
	log.Printf("registering handler for event: %s", event)
	d.handlers[event] = handler
}

func (d *Dispatcher) Dispatch(event string, ctx *wsstypes.WsContext) error {
	log.Printf("dispatching event: %s", event)

	handler, ok := d.handlers[event]
	if !ok {
		log.Printf("no handler found for event: %s", event)
		return errors.New("unknown event type: " + event)
	}

	err := handler(ctx)
	if err != nil {
		log.Printf("handler error for event %s: %v", event, err)
	}
	return err
}

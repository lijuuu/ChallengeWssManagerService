package wss

import (
	"errors"

	wsstypes "github.com/lijuuu/ChallengeWssManagerService/internal/wss/types"
)

// WsHandlerType defines the signature for a WebSocket event handler
type WsHandlerType func(*wsstypes.WsContext) error

// WsMiddleware defines the signature for middleware functions
type WsMiddleware func(*wsstypes.WsContext) error

// HandlerRegistration stores a handler with its associated middleware chain
type HandlerRegistration struct {
	Handler     WsHandlerType
	Middlewares []WsMiddleware
}

type Dispatcher struct {
	handlers map[string]*HandlerRegistration
}

func NewDispatcher() *Dispatcher {
	return &Dispatcher{
		handlers: make(map[string]*HandlerRegistration),
	}
}

func (d *Dispatcher) Register(event string, handler WsHandlerType) {
	d.handlers[event] = &HandlerRegistration{
		Handler:     handler,
		Middlewares: nil,
	}
}

func (d *Dispatcher) RegisterWithMiddleware(event string, handler WsHandlerType, middlewares ...WsMiddleware) {
	d.handlers[event] = &HandlerRegistration{
		Handler:     handler,
		Middlewares: middlewares,
	}
}

func (d *Dispatcher) Dispatch(event string, ctx *wsstypes.WsContext) error {
	registration, ok := d.handlers[event]
	if !ok {
		return errors.New("unknown event type: " + event)
	}

	// Execute middleware chain before handler
	if len(registration.Middlewares) > 0 {
		if err := d.executeMiddlewareChain(registration.Middlewares, ctx); err != nil {
			return err
		}
	}

	// Execute the main handler
	return registration.Handler(ctx)
}

// executeMiddlewareChain executes middleware functions in order, stopping on first error
func (d *Dispatcher) executeMiddlewareChain(middlewares []WsMiddleware, ctx *wsstypes.WsContext) error {
	for _, middleware := range middlewares {
		if err := middleware(ctx); err != nil {
			return err
		}
	}
	return nil
}

// Handler registration for PUSHNEWCHAT, PUSHNEWNOTIFICATION should be done in main setup:
// d.Register(constants.PUSH_NEW_CHAT, wsshandler.PushNewChatHandler)
// d.Register(constants.PUSH_NEW_CHAT, wsshandler.PushNewChatSyncHandler)
// d.Register(constants.PUSH_NEW_NOTIFICATION, wsshandler.PushNewNotificationHandler)

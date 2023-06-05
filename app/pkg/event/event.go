package event

import (
	"sync"
)

var (
	once          sync.Once
	eventsHandler *EventsHandler
)

type EventsHandler struct {
	mux        sync.RWMutex
	preProcess map[string]func(i interface{}) bool
	handlers   map[string]func(i interface{}) error
}

func NewEventsHandler() *EventsHandler {
	once.Do(func() {
		eventsHandler = &EventsHandler{
			mux:        sync.RWMutex{},
			preProcess: map[string]func(i interface{}) bool{},
			handlers:   map[string]func(i interface{}) error{},
		}
	})
	return eventsHandler
}

// RegHandler 注册handler
func (e *EventsHandler) RegHandler(t string, handler func(i interface{}) error) {
	e.mux.Lock()
	defer e.mux.Unlock()
	_, ok := e.handlers[t]
	if !ok {
		e.handlers[t] = handler
	}
}

// GetHandler 获取handler
func (e *EventsHandler) GetHandler(t string) func(i interface{}) error {
	e.mux.RLock()
	defer e.mux.RUnlock()
	handler, ok := e.handlers[t]
	if !ok {
		return nil
	} else {
		return handler
	}
}

// RegPreProcess 注册预处理PreProcess
func (e *EventsHandler) RegPreProcess(t string, preProcess func(i interface{}) bool) {
	e.mux.Lock()
	defer e.mux.Unlock()
	_, ok := e.preProcess[t]
	if !ok {
		e.preProcess[t] = preProcess
	}
}

// GetPreProcess 获取PreProcess
func (e *EventsHandler) GetPreProcess(t string) func(i interface{}) bool {
	e.mux.RLock()
	defer e.mux.RUnlock()
	preProcess, ok := e.preProcess[t]
	if !ok {
		return nil
	} else {
		return preProcess
	}
}

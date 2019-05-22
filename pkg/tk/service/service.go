package service

import (
	"context"
	"sync"

	"code.waarp.fr/waarp/gateway-ng/pkg/conf"
	"code.waarp.fr/waarp/gateway-ng/pkg/log"
)

type Name string

const (
	Admin Name = "Admin"
)

type Service interface {
	Start() error
	Stop(ctx context.Context) error
	State() *State
}

type Environment struct {
	*log.Logger
	Conf     *conf.ServerConfig
	Services map[Name]Service
}

func NewEnvironment(config *conf.ServerConfig) *Environment {
	return &Environment{
		Logger: log.NewLogger(),
		Conf:   config,
	}
}

type StateCode uint8

const (
	Offline StateCode = iota
	Starting
	Running
	ShuttingDown
	Error
)

func (s StateCode) Name() string {
	switch s {
	case Starting:
		return "Starting"
	case Running:
		return "Running"
	case ShuttingDown:
		return "Shutting down"
	case Error:
		return "Error"
	default:
		return "Offline"
	}
}

type State struct {
	code   StateCode
	reason string
	mutex  sync.RWMutex
}

func (s *State) Get() (StateCode, string) {
	defer s.mutex.RUnlock()
	s.mutex.RLock()
	return s.code, s.reason
}

func (s *State) Set(code StateCode, reason string) {
	defer s.mutex.Unlock()
	s.mutex.Lock()
	s.code = code
	s.reason = reason
}

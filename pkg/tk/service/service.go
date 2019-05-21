package service

import (
	"context"
	"sync"
)

type Service interface {
	Start() error
	Stop(ctx context.Context) error
}

type StateCode byte

const (
	DOWN StateCode = iota
	RUNNING
	ERROR
)

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
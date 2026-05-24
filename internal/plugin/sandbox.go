package plugin

import (
	"fmt"
)

type Sandbox struct {
	enabled   bool
	apis      map[string]bool
	memory    []byte
	cpuBudget int64
	netBudget int64
}

func NewSandbox(enabled bool, allowedAPIs []string) *Sandbox {
	apis := make(map[string]bool)
	for _, a := range allowedAPIs {
		apis[a] = true
	}
	return &Sandbox{
		enabled: enabled,
		apis:    apis,
	}
}

func (s *Sandbox) IsAPIAuthorized(name string) bool {
	if !s.enabled {
		return true
	}
	return s.apis[name]
}

func (s *Sandbox) SetMemoryLimit(bytes int64) {
	s.memory = make([]byte, bytes)
}

func (s *Sandbox) SetCPUBudget(cycles int64) {
	s.cpuBudget = cycles
}

func (s *Sandbox) SetNetworkBudget(bytes int64) {
	s.netBudget = bytes
}

func (s *Sandbox) Log(msg string) error {
	if !s.IsAPIAuthorized("log") {
		return fmt.Errorf("log API not authorized")
	}
	return nil
}

func (s *Sandbox) GetHeader(name string) (string, error) {
	if !s.IsAPIAuthorized("get_header") {
		return "", fmt.Errorf("get_header API not authorized")
	}
	return "", nil
}

func (s *Sandbox) SetHeader(name, value string) error {
	if !s.IsAPIAuthorized("set_header") {
		return fmt.Errorf("set_header API not authorized")
	}
	return nil
}

func (s *Sandbox) Block(reason string) error {
	if !s.IsAPIAuthorized("block") {
		return fmt.Errorf("block API not authorized")
	}
	return nil
}

func (s *Sandbox) Allow() error {
	if !s.IsAPIAuthorized("allow") {
		return fmt.Errorf("allow API not authorized")
	}
	return nil
}

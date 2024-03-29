package registry

import (
	"errors"
)

// The registry provides an interface for service discovery
// and an abstraction over varying implementations
// {consul, etcd, zookeeper, ...}
type Registry interface {
	Options() Options
	Register(*Service, ...RegisterOption) error
	Deregister(*Service) error
	GetService(string) ([]*Service, error)
	ListServices() ([]*Service, error)
	String() string
	Clean(typeName string) error
	Check(string) bool
}

type Option func(*Options)

type RegisterOption func(*RegisterOptions)

type WatchOption func(*WatchOptions)

var (
	DefaultRegistry = newRedisRegistry()
	ErrNotFound     = errors.New("not found")
)

func NewConsulRegistry(opts ...Option) Registry {
	return newConsulRegistry(opts...)
}

func NewRedisRegistry(opts ...Option) Registry {
	return newRedisRegistry(opts...)
}

// Register a service node. Additionally supply options such as TTL.
func Register(s *Service, opts ...RegisterOption) error {
	return DefaultRegistry.Register(s, opts...)
}

// Deregister a service node
func Deregister(s *Service) error {
	return DefaultRegistry.Deregister(s)
}

// Retrieve a service. A slice is returned since we separate Name/Version.
func GetService(name string) ([]*Service, error) {
	return DefaultRegistry.GetService(name)
}

// List the services. Only returns service names
func ListServices() ([]*Service, error) {
	return DefaultRegistry.ListServices()
}

func String() string {
	return DefaultRegistry.String()
}

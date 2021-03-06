package service

import (
	"sync"
	"time"

	"github.com/shihray/gserver/server"
	log "github.com/z9905080/gloger"
)

func NewService(opts ...Option) Service {
	return newService(opts...)
}

type service struct {
	opts Options

	once sync.Once
}

func newService(opts ...Option) Service {
	options := newOptions(opts...)

	return &service{
		opts: options,
	}
}

func (s *service) run(exit chan bool) {
	if s.opts.RegisterInterval <= time.Duration(0) {
		return
	}

	t := time.NewTicker(s.opts.RegisterInterval)

	for {
		select {
		case <-t.C:
			err := s.opts.Server.ServiceRegister()
			if err != nil {
				log.Warn("service run Server.Register error: ", err)
			}
		case <-exit:
			t.Stop()
			return
		}
	}
}

// Init initialises options. Additionally it calls cmd.Init
// which parses command line flags. cmd.Init is only called
// on first Init.
func (s *service) Init(opts ...Option) {
	// process options
	for _, o := range opts {
		o(&s.opts)
	}

	s.once.Do(func() {
		// save user action

	})
}

func (s *service) Options() Options {
	return s.opts
}

func (s *service) Server() server.Server {
	return s.opts.Server
}

func (s *service) String() string {
	return "gserver"
}

func (s *service) Start() error {
	if err := s.opts.Server.Start(); err != nil {
		return err
	}

	if err := s.opts.Server.ServiceRegister(); err != nil {
		return err
	}

	return nil
}

func (s *service) Stop() error {
	if err := s.opts.Server.ServiceDeregister(); err != nil {
		return err
	}

	if err := s.opts.Server.Stop(); err != nil {
		return err
	}
	return nil
}

func (s *service) Run() error {
	if err := s.Start(); err != nil {
		return err
	}

	// start reg loop
	ex := make(chan bool)
	go s.run(ex)

	select {
	// wait on kill signal
	case <-s.opts.Context.Done():
	}

	// exit reg loop
	close(ex)

	// return s.Stop()
	return nil
}

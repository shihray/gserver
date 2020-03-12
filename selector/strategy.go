package selector

import (
	"math/rand"
	"sync"
	"time"

	"github.com/shihray/gserver/registry"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

// Random is a random strategy algorithm for node selection
func Random(services []*registry.Service) Next {

	return func() (*registry.Service, error) {
		if len(services) == 0 {
			return nil, ErrNoneAvailable
		}

		i := rand.Int() % len(services)
		return services[i], nil
	}
}

// RoundRobin is a roundrobin strategy algorithm for node selection
func RoundRobin(services []*registry.Service) Next {

	var i = rand.Int()
	var mtx sync.Mutex

	return func() (*registry.Service, error) {
		if len(services) == 0 {
			return nil, ErrNoneAvailable
		}

		mtx.Lock()
		node := services[i%len(services)]
		i++
		mtx.Unlock()

		return node, nil
	}
}

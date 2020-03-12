// Package cache is a caching selector. It uses the registry watcher.
package cache

import (
	"sync"
	"time"

	"github.com/shihray/gserver/registry"
	"github.com/shihray/gserver/selector"
)

type cacheSelector struct {
	sync.RWMutex

	so selector.Options

	ttl  time.Duration
	ttls map[string]time.Time

	cache   map[string][]*registry.Service // registry cache
	watched map[string]bool                // service watch
	reload  chan bool                      // reload watcher
	exit    chan bool                      // close watcher
}

var (
	DefaultTTL = time.Minute
)

func (c *cacheSelector) quit() bool {
	select {
	case <-c.exit:
		return true
	default:
		return false
	}
}

// cp copies a service. Because we're caching handing back pointers would
// create a race condition, so we do this instead
// its fast enough
func (c *cacheSelector) cp(current []*registry.Service) []*registry.Service {
	var serviceList []*registry.Service

	for _, service := range current {
		// copy service
		s := new(registry.Service)
		*s = *service
		// append service
		serviceList = append(serviceList, s)
	}

	return serviceList
}

func (c *cacheSelector) del(service string) {
	delete(c.cache, service)
	delete(c.ttls, service)
}

func (c *cacheSelector) get(service string) ([]*registry.Service, error) {
	c.Lock()
	defer c.Unlock()

	// watch service if not watched
	if _, ok := c.watched[service]; !ok {
		go c.run(service)
		c.watched[service] = true
	}

	// get does the actual request for a service
	// it also caches it
	get := func(service string) ([]*registry.Service, error) {
		// ask the registry
		serviceList, err := c.so.Registry.GetService(service)
		if err != nil {
			return nil, err
		}

		// cache results
		c.set(service, c.cp(serviceList))
		return serviceList, nil
	}

	// check the cache first
	serviceList, ok := c.cache[service]

	// cache miss or no serviceList
	if !ok || len(serviceList) == 0 {
		return get(service)
	}

	// got cache but lets check ttl
	ttl, kk := c.ttls[service]

	// within ttl so return cache
	if kk && time.Since(ttl) < c.ttl {
		return c.cp(serviceList), nil
	}

	// expired entry so get service
	serviceList, err := get(service)

	// no error then return error
	if err == nil {
		return serviceList, nil
	}

	// not found error then return
	if err == registry.ErrNotFound {
		return nil, selector.ErrNotFound
	}

	// other error

	// return expired cache as last resort
	return c.cp(serviceList), nil
}

func (c *cacheSelector) set(service string, serviceList []*registry.Service) {
	c.cache[service] = serviceList
	c.ttls[service] = time.Now().Add(c.ttl)
}

func (c *cacheSelector) update(res *registry.Result) {
	if res == nil || res.Service == nil {
		return
	}

	c.Lock()
	defer c.Unlock()

	// existing service found
	var (
		serviceList []*registry.Service // multiple serviceList list
		service     *registry.Service   // signal service
		index       int                 // serviceList array index
		ok          bool                // found map key
	)

	// we're not going to cache anything, unless there was already a lookup
	if serviceList, ok = c.cache[res.Service.Name]; !ok {
		return
	}

	for i, s := range serviceList {
		if s.Name == res.Service.Name {
			service = s
			index = i
			break
		}
	}

	switch res.Action {
	case Create, Update:
		if service == nil {
			c.set(res.Service.Name, append(serviceList, res.Service))
			return
		}

		serviceList[index] = res.Service
		c.set(res.Service.Name, serviceList)
	case Delete: // filter cur nodes to remove the dead one
		if service == nil {
			return
		}
		var (
			iServiceList []*registry.Service
			seen         bool
		)
		for _, cur := range serviceList {
			if service.ID == cur.ID {
				seen = true
				break
			}
			if !seen {
				iServiceList = append(iServiceList, cur)
			} else {
				//应该删除的
				if c.Options().Watcher != nil {
					c.Options().Watcher(cur)
				}
			}
		}

		// still got nodes, save and return
		if len(iServiceList) > 0 {
			serviceList[index] = service
			c.set(service.Name, serviceList)
			return
		}

		// zero nodes left

		// only have one thing to delete
		// nuke the thing
		if len(serviceList) == 1 {
			c.del(service.Name)
			return
		}

		// still have more than 1 service
		// check the version and keep what we know
		var srvs []*registry.Service
		for _, s := range serviceList {
			if s.Name != service.Name {
				srvs = append(srvs, s)
			}
		}

		// save
		c.set(service.Name, srvs)
	}
}

// run starts the cache watcher loop
// it creates a new watcher if there's a problem
// reloads the watcher if Init is called
// and returns when Close is called
func (c *cacheSelector) run(name string) {
	// for {
	// 	// exit early if already dead
	// 	if c.quit() {
	// 		return
	// 	}

	// 	// create new watcher
	// 	w, err := c.so.Registry.Watch(
	// 		registry.WatchService(name),
	// 	)
	// 	if err != nil {
	// 		if c.quit() {
	// 			return
	// 		}
	// 		logging.Warn("%v", err)
	// 		time.Sleep(time.Second)
	// 		continue
	// 	}

	// 	// watch for events
	// 	if err := c.watch(w); err != nil {
	// 		if c.quit() {
	// 			return
	// 		}
	// 		logging.Warn("%v", err)
	// 		continue
	// 	}
	// }
}

// watch loops the next event and calls update
// it returns if there's an error
func (c *cacheSelector) watch(w registry.Watcher) error {
	defer w.Stop()

	// manage this loop
	go func() {
		// wait for exit or reload signal
		select {
		case <-c.exit:
		case <-c.reload:
		}

		// stop the watcher
		w.Stop()
	}()

	for {
		res, err := w.Next()
		if err != nil {
			return err
		}
		c.update(res)
	}
}

func (c *cacheSelector) Init(opts ...selector.Option) error {
	for _, o := range opts {
		o(&c.so)
	}

	// reload the watcher
	go func() {
		select {
		case <-c.exit:
			return
		default:
			c.reload <- true
		}
	}()

	return nil
}

func (c *cacheSelector) Options() selector.Options {
	return c.so
}

func (c *cacheSelector) GetService(service string) ([]*registry.Service, error) {
	serviceList, err := c.get(service)
	if err != nil {
		return nil, err
	}
	return serviceList, nil
}

func (c *cacheSelector) Select(service string, opts ...selector.SelectOption) (selector.Next, error) {
	sopts := selector.SelectOptions{
		Strategy: c.so.Strategy,
	}

	for _, opt := range opts {
		opt(&sopts)
	}

	// get the service
	// try the cache first
	// if that fails go directly to the registry
	serviceList, err := c.get(service)
	if err != nil {
		return nil, err
	}

	// apply the filters
	for _, filter := range sopts.Filters {
		serviceList = filter(serviceList)
	}

	// if there's nothing left, return
	if len(serviceList) == 0 {
		return nil, selector.ErrNoneAvailable
	}

	return sopts.Strategy(serviceList), nil
}

func (c *cacheSelector) Mark(service string, s *registry.Service, err error) {
	return
}

func (c *cacheSelector) Reset(service string) {
	return
}

// Close stops the watcher and destroys the cache
func (c *cacheSelector) Close() error {
	c.Lock()
	c.cache = make(map[string][]*registry.Service)
	c.watched = make(map[string]bool)
	c.Unlock()

	select {
	case <-c.exit:
		return nil
	default:
		close(c.exit)
	}
	return nil
}

func (c *cacheSelector) String() string {
	return "cache"
}

func NewSelector(opts ...selector.Option) selector.Selector {
	sopts := selector.Options{
		Strategy: selector.Random,
	}

	for _, opt := range opts {
		opt(&sopts)
	}

	if sopts.Registry == nil {
		sopts.Registry = registry.DefaultRegistry
	}

	ttl := DefaultTTL

	if sopts.Context != nil {
		if t, ok := sopts.Context.Value(ttlKey{}).(time.Duration); ok {
			ttl = t
		}
	}

	return &cacheSelector{
		so:      sopts,
		ttl:     ttl,
		watched: make(map[string]bool),
		cache:   make(map[string][]*registry.Service),
		ttls:    make(map[string]time.Time),
		reload:  make(chan bool, 1),
		exit:    make(chan bool),
	}
}

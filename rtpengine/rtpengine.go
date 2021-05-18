package rtpengine

import (
	"context"
	"strconv"
	"sync"

	"github.com/rotisserie/eris"
	"go.uber.org/zap"
	discoveryv1 "k8s.io/api/discovery/v1"
	"k8s.io/client-go/tools/cache"
)

// Set represents an RTPEngine set
type Set struct {
	ID int

	ServiceName string
	ServiceNamespace string
	ServicePort string

	Endpoints []*Endpoint

	Informer cache.SharedIndexInformer

	Logger *zap.Logger

	changed chan struct{}
	mu sync.Mutex
}

// Endpoint describes an RTPEngine control Endpoint
type Endpoint struct {
	Address string
	Port uint32
}

func (s *Set) Start() error {
	if s.ServiceName == "" {
		return eris.New("ServiceName must be set")
	}

	if s.ServiceNamespace == "" {
		return eris.New("ServiceNamespace must be set")
	}

	if s.ServicePort == "" {
		return eris.New("ServicePort must be set")
	}

	if s.Informer == nil {
		return eris.New("Informer must be set")
	}

	if s.Logger == nil {
		return eris.New("Logger must be set")
	}

	s.changed = make(chan struct{},1)

	s.Informer.AddEventHandler(s)

	return nil
}

func matchService(ns, name string, o *discoveryv1.EndpointSlice) bool {
	if o.Name != name {
		return false
	}

	return o.Namespace == ns
}

// OnAdd implements cache.ResourceEventHandler
func (s *Set) OnAdd(obj interface{}) {
	epSlice, ok := obj.(*discoveryv1.EndpointSlice)
	if !ok {
		return
	}

	if !matchService(s.ServiceNamespace, s.ServiceName, epSlice) {
		return
	}

	s.update(epSlice)
}

// OnUpdate implements cache.ResourceEventHandler
func (s *Set) OnUpdate(oldObj interface{}, newObj interface{}) {
	epSlice, ok := newObj.(*discoveryv1.EndpointSlice)
	if !ok {
		return
	}

	if !matchService(s.ServiceNamespace, s.ServiceName, epSlice) {
		return
	}

	s.update(epSlice)
}

// OnDelete implements cache.ResourceEventHandler
func (s *Set) OnDelete(obj interface{}) {
	epSlice, ok := obj.(*discoveryv1.EndpointSlice)
	if !ok {
		return
	}

	if !matchService(s.ServiceNamespace, s.ServiceName, epSlice) {
		return
	}

	s.update(nil)
}



// Watch monitors the Service for changes, updating the Endpoints and returning
// when changes are detected.
func (s *Set) Watch(ctx context.Context) (err error) {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-s.changed:
			return nil
		}
}

func (s *Set) update(epSlice *discoveryv1.EndpointSlice) {
	if epSlice == nil {
		if s.Endpoints != nil {
			s.Endpoints = nil

			select {
			case s.changed <- struct{}{}:
			default:
			}
		}

		return
	}

	current, err := flattenEndpointSlice(s.ServicePort, epSlice)
	if err != nil {
		return
	}

	if isChanged(s.Endpoints, current) {
		s.Endpoints = current
		
		select {
		case s.changed <- struct{}{}:
		default:
		}
	}

	return
}

func isChanged(previous []*Endpoint, current []*Endpoint) (changed bool) {
	if len(previous) != len(current) {
		return true
	}

	for _, p := range previous {
		var found bool

		for _, c := range current {
			if c.Address == p.Address &&
				c.Port == p.Port {
				found = true
				break
			}
		}

		if !found {
			return false
		}
	}

	return true
}

func flattenEndpointSlice(refPort string, epSlice *discoveryv1.EndpointSlice) (out []*Endpoint, err error) {
	portNumber, err := strconv.Atoi(refPort)
	if err != nil {
		portNumber = 0
	}

	if portNumber == 0 {
		for _, p := range epSlice.Ports {
			if p.Name == nil {
				continue
			}

			if *p.Name == refPort {
				if p.Port == nil {
					return nil, eris.Errorf("endpoint port %s has no numerical port", p.Name)
				}
				portNumber = int(*p.Port)
			}
		}

		if portNumber == 0 {
			return nil, eris.Errorf("failed to find port %s in EndpointSlice %s", refPort, epSlice.Name)
		}
	}


	for _, n := range epSlice.Endpoints {
		for _, addr := range n.Addresses {
			out = append(out, &Endpoint{
				Address: addr,
				Port: uint32(portNumber),
			})
		}
	}

	return out, nil
}


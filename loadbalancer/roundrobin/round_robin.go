package roundrobin

import (
	"context"
	"sync"
	"time"

	"github.com/failsafe-go/failsafe-go/circuitbreaker"
	"github.com/relychan/gohttpc/loadbalancer"
	"github.com/relychan/goutils"
)

// WeightedRoundRobin represents the load balancer for
// Weighted Round-Robin algorithm implementation.
type WeightedRoundRobin struct {
	lock                sync.Mutex
	servers             []*loadbalancer.Server
	isSameWeight        bool
	totalWeight         int
	tick                *time.Ticker
	healthCheckInterval time.Duration
}

var _ loadbalancer.LoadBalancer = (*WeightedRoundRobin)(nil)

// NewWeightedRoundRobin method creates the new Weighted Round-Robin
// load balancer instance with given recovery duration and hosts slice.
func NewWeightedRoundRobin(
	healthCheckInterval time.Duration,
	servers []*loadbalancer.Server,
) (*WeightedRoundRobin, error) {
	wrr := &WeightedRoundRobin{
		healthCheckInterval: healthCheckInterval,
	}

	err := wrr.Refresh(servers)

	return wrr, err
}

// Next returns the next server based on the Weighted Round-Robin algorithm.
func (wrr *WeightedRoundRobin) Next() (*loadbalancer.Server, error) {
	wrr.lock.Lock()
	defer wrr.lock.Unlock()

	if wrr.isSameWeight {
		return wrr.nextRoundRobin()
	}

	return wrr.nextWeightRoundRobin()
}

// Refresh resets the existing values with the given [Host] slice to refresh it.
func (wrr *WeightedRoundRobin) Refresh(servers []*loadbalancer.Server) error {
	if servers == nil {
		return nil
	}

	wrr.lock.Lock()
	defer wrr.lock.Unlock()

	isSameWeight := true
	lastWeight := 0
	newTotalWeight := 0

	for i, h := range servers {
		weight := h.Weight()
		newTotalWeight += h.Weight()

		if i == 0 {
			lastWeight = weight
		} else if isSameWeight && lastWeight != weight {
			isSameWeight = false
		}
	}

	// after processing, assign the updates
	wrr.servers = servers
	wrr.isSameWeight = isSameWeight

	if isSameWeight {
		// Start the round robin algorithm since all weight are the same.
		// Reuse the totalWeight as the current index.
		wrr.totalWeight = 0
	} else {
		wrr.totalWeight = newTotalWeight
	}

	return nil
}

// Close method does the cleanup by stopping the [time.Ticker] on the load balancer.
func (wrr *WeightedRoundRobin) Close() error {
	wrr.lock.Lock()
	defer wrr.lock.Unlock()

	if wrr.tick == nil {
		return nil
	}

	wrr.tick.Stop()
	wrr.tick = nil

	for _, server := range wrr.servers {
		server.Close()
	}

	return nil
}

// Servers return the list of server of the load balancer.
func (wrr *WeightedRoundRobin) Servers() []*loadbalancer.Server {
	wrr.lock.Lock()
	defer wrr.lock.Unlock()

	return wrr.servers
}

// StartHealthCheck starts a ticker to run health checking for servers in the background.
func (wrr *WeightedRoundRobin) StartHealthCheck(ctx context.Context) {
	if wrr.healthCheckInterval <= 0 {
		return
	}

	if wrr.tick != nil {
		goutils.CatchWarnErrorFunc(wrr.Close)
	}

	wrr.tick = time.NewTicker(wrr.healthCheckInterval)

	for {
		select {
		case <-ctx.Done():
			goutils.CatchWarnErrorFunc(wrr.Close)

			return
		case <-wrr.tick.C:
			for _, host := range wrr.servers {
				host.CheckHealth(ctx)
			}
		}
	}
}

// the next server based on the Round-Robin algorithm.
func (rr *WeightedRoundRobin) nextRoundRobin() (*loadbalancer.Server, error) {
	totalServers := len(rr.servers)

	for i := range totalServers {
		currentIndex := (i + rr.totalWeight) % totalServers
		server := rr.servers[currentIndex]

		if server.State() != circuitbreaker.OpenState {
			rr.totalWeight = (currentIndex + 1) % totalServers

			return server, nil
		}
	}

	return nil, loadbalancer.ErrNoActiveHost
}

// Find the next server based on the Weighted Round-Robin algorithm.
func (wrr *WeightedRoundRobin) nextWeightRoundRobin() (*loadbalancer.Server, error) {
	var best *loadbalancer.Server

	total := 0

	for _, h := range wrr.servers {
		if h.State() == circuitbreaker.OpenState {
			continue
		}

		h.AddCurrentWeight()

		total += h.Weight()

		if best == nil || h.CurrentWeight() > best.CurrentWeight() {
			best = h
		}
	}

	if best == nil {
		return nil, loadbalancer.ErrNoActiveHost
	}

	best.ResetCurrentWeight(total)

	return best, nil
}

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
	weightedRoundRobinOptions

	lock         sync.Mutex
	hosts        []*loadbalancer.Host
	isSameWeight bool
	totalWeight  int
	tick         *time.Ticker
}

var _ loadbalancer.LoadBalancer = (*WeightedRoundRobin)(nil)

// NewWeightedRoundRobin method creates the new Weighted Round-Robin
// load balancer instance with given recovery duration and hosts slice.
func NewWeightedRoundRobin(
	hosts []*loadbalancer.Host,
	options ...WeightedRoundRobinOption,
) (*WeightedRoundRobin, error) {
	wrr := &WeightedRoundRobin{
		weightedRoundRobinOptions: weightedRoundRobinOptions{
			healthCheckInterval: 0,
		},
	}

	for _, opt := range options {
		opt(&wrr.weightedRoundRobinOptions)
	}

	err := wrr.Refresh(hosts)

	return wrr, err
}

// Next returns the next server based on the Weighted Round-Robin algorithm.
func (wrr *WeightedRoundRobin) Next() (*loadbalancer.Host, error) {
	wrr.lock.Lock()
	defer wrr.lock.Unlock()

	switch len(wrr.hosts) {
	case 0:
		return nil, loadbalancer.ErrNoActiveHost
	case 1:
		// Return the only host directly.
		return wrr.hosts[0], nil
	default:
		if wrr.isSameWeight {
			return wrr.nextRoundRobin(), nil
		}

		return wrr.nextWeightRoundRobin(), nil
	}
}

// Refresh resets the existing values with the given [Host] slice to refresh it.
func (wrr *WeightedRoundRobin) Refresh(servers []*loadbalancer.Host) error {
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

		hcPolicy := h.HealthCheckPolicy()
		if hcPolicy == nil {
			continue
		}
	}

	// after processing, assign the updates
	wrr.hosts = servers
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

	for _, host := range wrr.hosts {
		host.Close()
	}

	return nil
}

// Hosts return the list of hosts of the load balancer.
func (wrr *WeightedRoundRobin) Hosts() []*loadbalancer.Host {
	wrr.lock.Lock()
	defer wrr.lock.Unlock()

	return wrr.hosts
}

// StartHealthCheck starts a ticker to run health checking for servers in the background.
func (wrr *WeightedRoundRobin) StartHealthCheck(ctx context.Context) {
	wrr.lock.Lock()

	if wrr.healthCheckInterval <= 0 {
		wrr.lock.Unlock()

		return
	}

	if wrr.tick != nil {
		goutils.CatchWarnErrorFunc(wrr.Close)
	}

	newTicker := time.NewTicker(wrr.healthCheckInterval)
	wrr.tick = newTicker

	wrr.lock.Unlock()

	for {
		select {
		case <-ctx.Done():
			goutils.CatchWarnErrorFunc(wrr.Close)

			return
		case <-newTicker.C:
			for _, host := range wrr.Hosts() {
				host.CheckHealth(ctx)
			}
		}
	}
}

// the next server based on the Round-Robin algorithm.
func (rr *WeightedRoundRobin) nextRoundRobin() *loadbalancer.Host {
	totalServers := len(rr.hosts)

	var fallbackHost *loadbalancer.Host

	for i := range totalServers {
		currentIndex := (i + rr.totalWeight) % totalServers
		server := rr.hosts[currentIndex]

		policy := server.HealthCheckPolicy()
		if policy != nil {
			if policy.State() == circuitbreaker.OpenState {
				// checks if the open state was expired.
				if !policy.TryAcquirePermit() {
					_, isOutage := server.GetLastHTTPErrorStatus()
					if !isOutage {
						fallbackHost = server
					}

					continue
				}
			}
		}

		rr.totalWeight = (currentIndex + 1) % totalServers

		return server
	}

	if fallbackHost == nil {
		fallbackHost = rr.hosts[rr.totalWeight]
	}

	rr.totalWeight = (rr.totalWeight + 1) % totalServers

	return fallbackHost
}

// Find the next server based on the Weighted Round-Robin algorithm.
func (wrr *WeightedRoundRobin) nextWeightRoundRobin() *loadbalancer.Host {
	var best, fallbackHost *loadbalancer.Host

	total := 0

	for _, h := range wrr.hosts {
		policy := h.HealthCheckPolicy()
		if policy != nil {
			if policy.State() == circuitbreaker.OpenState {
				// checks if the open state is expired.
				if !policy.TryAcquirePermit() {
					_, isOutage := h.GetLastHTTPErrorStatus()
					if !isOutage {
						fallbackHost = h
					}

					continue
				}
			}
		}

		h.AddCurrentWeight()

		total += h.Weight()

		if best == nil || h.CurrentWeight() > best.CurrentWeight() {
			best = h
		}
	}

	if best != nil {
		best.ResetCurrentWeight(total)

		return best
	}

	if fallbackHost == nil {
		fallbackHost = wrr.hosts[0]
	}

	return fallbackHost
}

type weightedRoundRobinOptions struct {
	healthCheckInterval time.Duration
}

// WeightedRoundRobinOption represents a function to modify the Weight Round-Robin options.
type WeightedRoundRobinOption func(*weightedRoundRobinOptions)

// WithHealthCheckInterval sets the health check interval for the round robin.
func WithHealthCheckInterval(duration time.Duration) WeightedRoundRobinOption {
	return func(wrro *weightedRoundRobinOptions) {
		wrro.healthCheckInterval = duration
	}
}

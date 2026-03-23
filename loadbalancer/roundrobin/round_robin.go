// Copyright 2026 RelyChan Pte. Ltd
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
	tick         *time.Ticker
	isSameWeight bool
	totalWeight  uint16
}

var _ loadbalancer.LoadBalancer = (*WeightedRoundRobin)(nil)

// NewWeightedRoundRobin creates a new Weighted Round-Robin
// load balancer instance with the given hosts slice and optional configuration.
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

	var newTotalWeight, lastWeight uint16

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
	if wrr.healthCheckInterval <= 0 {
		return
	}

	if wrr.tick != nil {
		goutils.CatchWarnErrorFunc(wrr.Close)
	}

	newTicker := time.NewTicker(wrr.healthCheckInterval)

	wrr.lock.Lock()
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

// Returns the next server based on the Round-Robin algorithm.
func (rr *WeightedRoundRobin) nextRoundRobin() *loadbalancer.Host {
	totalServers := uint16(len(rr.hosts))

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

// nextWeightRoundRobin returns the next server based on the Weighted Round-Robin algorithm.
func (wrr *WeightedRoundRobin) nextWeightRoundRobin() *loadbalancer.Host {
	var best, fallbackHost *loadbalancer.Host

	var total uint16

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

// WeightedRoundRobinOption represents a function to modify the Weighted Round-Robin options.
type WeightedRoundRobinOption func(*weightedRoundRobinOptions)

// WithHealthCheckInterval sets the health check interval for the round robin.
func WithHealthCheckInterval(duration time.Duration) WeightedRoundRobinOption {
	return func(wrro *weightedRoundRobinOptions) {
		wrro.healthCheckInterval = max(
			// Negative durations are not allowed; set to zero (or could ignore assignment)
			duration, 0)
	}
}

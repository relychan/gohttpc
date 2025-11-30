package loadbalancer

import (
	"context"
	"errors"

	"github.com/relychan/gohttpc"
)

// ErrNoActiveHost occurs when all hosts are inactive on the load balancer.
var ErrNoActiveHost = errors.New("no active host")

// LoadBalancer is the interface that wraps the HTTP client load-balancing
// algorithm that returns the appropriate host for the request to target.
type LoadBalancer interface {
	Next() (*Server, error)
	// StartHealthCheck starts a ticker to run health checking for servers in the background.
	StartHealthCheck(ctx context.Context)
	Close() error
}

// LoadBalancerClient represents an HTTP client that accepts a list of hosts
// and load balance requests to each host.
type LoadBalancerClient struct {
	loadBalancer LoadBalancer
	options      *gohttpc.ClientOptions
}

// NewLoadBalancerClient creates a new [LoadBalancerClient] instance.
func NewLoadBalancerClient(
	loadBalancer LoadBalancer,
	options *gohttpc.ClientOptions,
) *LoadBalancerClient {
	return &LoadBalancerClient{
		loadBalancer: loadBalancer,
		options:      options,
	}
}

// R is the shortcut to create a Request given a method, URL with default request options.
func (lbc *LoadBalancerClient) R(method string, url string) *gohttpc.RequestWithClient {
	return gohttpc.NewRequestWithClient(
		gohttpc.NewRequest(method, url, &lbc.options.RequestOptions),
		lbc,
	)
}

// HTTPClient returns the current or inner HTTP client for load balancing.
func (lbc *LoadBalancerClient) HTTPClient() (gohttpc.HTTPClient, error) {
	return lbc.loadBalancer.Next()
}

// StartHealthCheck starts a ticker to run health checking for servers in the background.
func (lbc *LoadBalancerClient) StartHealthCheck(ctx context.Context) {
	if lbc.loadBalancer == nil {
		return
	}

	lbc.loadBalancer.StartHealthCheck(ctx)
}

// Close terminates the client and clean up internal processes.
func (lbc *LoadBalancerClient) Close() error {
	if lbc.loadBalancer == nil {
		return nil
	}

	return lbc.loadBalancer.Close()
}

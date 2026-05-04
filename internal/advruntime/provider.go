package advruntime

import (
	"context"
	"errors"
	"sync"

	operatorservice "go.temporal.io/api/operatorservice/v1"
	workflowservice "go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/client"
)

var ErrProviderClosed = errors.New("temporal client provider closed")

// TemporalClient is the narrow subset of Temporal SDK client.Client used by
// ADV runtime health. Keeping this interface small makes classifier tests pure
// Go and avoids requiring a live Temporal server.
type TemporalClient interface {
	WorkflowService() workflowservice.WorkflowServiceClient
	OperatorService() operatorservice.OperatorServiceClient
	Close()
}

type DialFunc func(context.Context, client.Options) (TemporalClient, error)

func DefaultTemporalDialer(ctx context.Context, opts client.Options) (TemporalClient, error) {
	return client.DialContext(ctx, opts)
}

type ClientProvider struct {
	config Config
	dial   DialFunc

	mu     sync.Mutex
	client TemporalClient
	closed bool
}

func NewTemporalClientProvider(config Config, dial DialFunc) *ClientProvider {
	if dial == nil {
		dial = DefaultTemporalDialer
	}
	return &ClientProvider{config: config.WithDefaults(), dial: dial}
}

func (p *ClientProvider) Client(ctx context.Context) (TemporalClient, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return nil, ErrProviderClosed
	}
	if p.client != nil {
		return p.client, nil
	}

	opts := client.Options{
		HostPort:  p.config.Address,
		Namespace: p.config.Namespace,
	}
	c, err := p.dial(ctx, opts)
	if err != nil {
		return nil, err
	}
	p.client = c
	return c, nil
}

func (p *ClientProvider) Close() {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return
	}
	p.closed = true
	if p.client != nil {
		p.client.Close()
		p.client = nil
	}
}

type FakeClientProvider struct {
	ClientValue TemporalClient
	Err         error
}

func NewFakeTemporalClientProvider(client TemporalClient, err error) *FakeClientProvider {
	return &FakeClientProvider{ClientValue: client, Err: err}
}

func (p *FakeClientProvider) Client(context.Context) (TemporalClient, error) {
	if p.Err != nil {
		return nil, p.Err
	}
	return p.ClientValue, nil
}

func (p *FakeClientProvider) Close() {}

package sseserver

import "time"

const (
	defaultConnectionBuffer = 256
	defaultPublishBuffer    = 256
	defaultKeepAlive        = 15 * time.Second
)

type config struct {
	connectionBuffer int
	publishBuffer    int
	keepAlive        time.Duration
}

func defaultConfig() config {
	return config{
		connectionBuffer: defaultConnectionBuffer,
		publishBuffer:    defaultPublishBuffer,
		keepAlive:        defaultKeepAlive,
	}
}

// Option customizes a Server.
type Option func(*config)

// WithConnectionBuffer sets the per-connection outbound queue size.
func WithConnectionBuffer(size int) Option {
	return func(cfg *config) {
		if size > 0 {
			cfg.connectionBuffer = size
		}
	}
}

// WithPublishBuffer sets the size of the internal publish queue.
func WithPublishBuffer(size int) Option {
	return func(cfg *config) {
		if size > 0 {
			cfg.publishBuffer = size
		}
	}
}

// WithKeepAliveInterval sets the SSE keepalive interval.
func WithKeepAliveInterval(interval time.Duration) Option {
	return func(cfg *config) {
		if interval > 0 {
			cfg.keepAlive = interval
		}
	}
}

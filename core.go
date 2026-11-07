package logging

import (
	"sync"
	"sync/atomic"
)

// Core is the log write pipeline (async queue plus JSON encoding sink).
type Core interface {
	Write(item *LogItem, encrypt bool) error
	Close() error
}

var (
	coreMu      sync.RWMutex
	defaultCore Core
	// DroppedLogs counts log lines dropped when the async queue is full (monitoring hook).
	DroppedLogs atomic.Uint64
	// WrittenLogs counts log lines successfully written after dequeue.
	WrittenLogs atomic.Uint64
)

// Close drains the async queue and shuts down workers; safe to call multiple times before exit.
func Close() error {
	coreMu.Lock()
	defer coreMu.Unlock()
	return closeCoreLocked()
}

func closeCoreLocked() error {
	if defaultCore == nil {
		IsSetup = false
		return nil
	}
	err := defaultCore.Close()
	defaultCore = nil
	IsSetup = false
	return err
}

func writeLog(item *LogItem, encrypt bool) {
	coreMu.RLock()
	c := defaultCore
	coreMu.RUnlock()
	if c == nil {
		return
	}
	_ = c.Write(item, encrypt)
}

func buildCore(cfg Config) (Core, error) {
	sink, err := newEncodeSink(cfg)
	if err != nil {
		return nil, err
	}
	if cfg.AsyncSync || cfg.Output == OutputFmt {
		return sink, nil
	}
	size := cfg.AsyncQueueSize
	if size <= 0 {
		size = defaultAsyncQueueSize
	}
	return newAsyncCore(sink, size, !cfg.AsyncBlockWhenFull), nil
}

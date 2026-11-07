package logging

import (
	"encoding/json"
	"sync"
)

type logTask struct {
	item    *LogItem
	encrypt bool
}

type asyncCore struct {
	next        Core
	ch          chan *logTask
	dropFull    bool
	wg          sync.WaitGroup
	once        sync.Once
	mu          sync.Mutex
	closed      bool
	closeNotify chan struct{}
}

func newAsyncCore(next Core, queueSize int, dropFull bool) *asyncCore {
	a := &asyncCore{
		next:        next,
		ch:          make(chan *logTask, queueSize),
		dropFull:    dropFull,
		closeNotify: make(chan struct{}),
	}
	a.wg.Add(1)
	go a.worker()
	return a
}

func (a *asyncCore) Write(item *LogItem, encrypt bool) error {
	task, err := cloneLogTask(item, encrypt)
	if err != nil {
		return err
	}
	if a.dropFull {
		a.mu.Lock()
		if a.closed {
			a.mu.Unlock()
			return nil
		}
		select {
		case a.ch <- task:
		default:
			if !a.closed {
				DroppedLogs.Add(1)
			}
		}
		a.mu.Unlock()
		return nil
	}
	for {
		a.mu.Lock()
		if a.closed {
			a.mu.Unlock()
			return nil
		}
		a.mu.Unlock()
		select {
		case a.ch <- task:
			return nil
		case <-a.closeNotify:
		}
	}
}

func (a *asyncCore) Close() error {
	a.once.Do(func() {
		a.mu.Lock()
		a.closed = true
		a.mu.Unlock()
		close(a.closeNotify)
		close(a.ch)
	})
	a.wg.Wait()
	if c, ok := a.next.(interface{ Close() error }); ok {
		return c.Close()
	}
	return nil
}

func (a *asyncCore) worker() {
	defer a.wg.Done()
	for task := range a.ch {
		_ = a.next.Write(task.item, task.encrypt)
		WrittenLogs.Add(1)
	}
}

func cloneLogTask(item *LogItem, encrypt bool) (*logTask, error) {
	b, err := json.Marshal(item)
	if err != nil {
		return nil, err
	}
	var copy LogItem
	if err := json.Unmarshal(b, &copy); err != nil {
		return nil, err
	}
	return &logTask{item: &copy, encrypt: encrypt}, nil
}

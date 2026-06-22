// 同一キーの非同期タスクを直列化し、実行中の再要求を1回に畳み込む仕組みを提供する。
package app

import "sync"

// asyncCoalescer は同一キー（gameID）のタスクを直列実行し、実行中に来た再要求を
// 完了後に1回だけ再実行する（coalescing）。異なるキーは並行に実行される。
type asyncCoalescer struct {
	mu       sync.Mutex
	inFlight map[string]bool
	pending  map[string]bool
	run      func(id string)
}

// newAsyncCoalescer は run を実行関数とする asyncCoalescer を生成する。
func newAsyncCoalescer(run func(id string)) *asyncCoalescer {
	return &asyncCoalescer{
		inFlight: make(map[string]bool),
		pending:  make(map[string]bool),
		run:      run,
	}
}

// trigger は id のタスクを要求する。既に実行中なら完了後の再実行をマークするだけで即座に返る。
func (c *asyncCoalescer) trigger(id string) {
	c.mu.Lock()
	if c.inFlight[id] {
		c.pending[id] = true
		c.mu.Unlock()
		return
	}
	c.inFlight[id] = true
	c.mu.Unlock()
	go c.loop(id)
}

// loop は id のタスクを実行し、実行中に再要求があれば1回だけ追加実行してから終了する。
func (c *asyncCoalescer) loop(id string) {
	for {
		c.run(id)
		c.mu.Lock()
		if c.pending[id] {
			delete(c.pending, id)
			c.mu.Unlock()
			continue
		}
		delete(c.inFlight, id)
		c.mu.Unlock()
		return
	}
}

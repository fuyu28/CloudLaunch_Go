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
	// onPanic は run が panic を起こした際に回収した値とともに呼ばれる（任意）。
	onPanic func(id string, recovered any)
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
		if c.runSafely(id) {
			// run が panic した場合は再実行せず（pending を消化すると panic ループに
			// なりうる）、inFlight を確実にクリアして終了する。次回 trigger で再開できる。
			c.mu.Lock()
			delete(c.pending, id)
			delete(c.inFlight, id)
			c.mu.Unlock()
			return
		}
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

// runSafely は run を実行し、panic を回収する。panic したら true を返し、loop 側で
// inFlight を確実にクリアして同期が永久にブロックされるのを防ぐ。
func (c *asyncCoalescer) runSafely(id string) (panicked bool) {
	defer func() {
		if r := recover(); r != nil {
			panicked = true
			if c.onPanic != nil {
				c.onPanic(id, r)
			}
		}
	}()
	c.run(id)
	return false
}

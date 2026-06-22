package app

import (
	"sync"
	"testing"
	"time"
)

// TestAsyncCoalescerCoalescesWhileInFlight は、実行中に来た複数の再要求が
// 完了後の1回の再実行に畳み込まれることを確認する。
func TestAsyncCoalescerCoalescesWhileInFlight(t *testing.T) {
	calledCh := make(chan int, 16)
	release := make(chan struct{})
	var mu sync.Mutex
	calls := 0
	c := newAsyncCoalescer(func(_ string) {
		mu.Lock()
		calls++
		n := calls
		mu.Unlock()
		calledCh <- n
		if n == 1 {
			<-release // 1回目を実行中のまま保持し、その間に再要求を畳み込ませる
		}
	})

	c.trigger("g1")
	if got := <-calledCh; got != 1 {
		t.Fatalf("first run expected 1, got %d", got)
	}
	// 実行中に3回再要求 → 1回に畳み込まれるはず
	c.trigger("g1")
	c.trigger("g1")
	c.trigger("g1")
	close(release)

	select {
	case got := <-calledCh:
		if got != 2 {
			t.Fatalf("coalesced run expected 2, got %d", got)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("coalesced second run did not happen")
	}

	// 3回目以降は起きない
	select {
	case got := <-calledCh:
		t.Fatalf("unexpected extra run: %d", got)
	case <-time.After(200 * time.Millisecond):
	}
}

// TestAsyncCoalescerRunsOncePerTriggerWhenSequential は、逐次的な trigger では
// 各要求がそれぞれ実行されることを確認する（畳み込みは実行中の場合のみ）。
func TestAsyncCoalescerRunsOncePerTriggerWhenSequential(t *testing.T) {
	doneCh := make(chan struct{}, 8)
	c := newAsyncCoalescer(func(_ string) {
		doneCh <- struct{}{}
	})

	c.trigger("g1")
	select {
	case <-doneCh:
	case <-time.After(2 * time.Second):
		t.Fatal("first run did not happen")
	}
	// 1回目完了後に再 trigger → これも実行される
	c.trigger("g1")
	select {
	case <-doneCh:
	case <-time.After(2 * time.Second):
		t.Fatal("second sequential run did not happen")
	}
}

// TestAsyncCoalescerDifferentKeysRunConcurrently は異なるキーが並行実行されることを確認する。
func TestAsyncCoalescerDifferentKeysRunConcurrently(t *testing.T) {
	bothStarted := make(chan struct{})
	var once sync.Once
	started := make(chan string, 2)
	hold := make(chan struct{})
	c := newAsyncCoalescer(func(id string) {
		started <- id
		<-hold
	})
	c.trigger("a")
	c.trigger("b")
	got := map[string]bool{}
	for i := 0; i < 2; i++ {
		select {
		case id := <-started:
			got[id] = true
		case <-time.After(2 * time.Second):
			t.Fatal("both keys did not start concurrently")
		}
	}
	once.Do(func() { close(bothStarted) })
	close(hold)
	if !got["a"] || !got["b"] {
		t.Fatalf("expected both a and b to start, got %v", got)
	}
}

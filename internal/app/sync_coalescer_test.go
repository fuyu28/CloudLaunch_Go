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

// TestAsyncCoalescerRecoversFromPanic は、run が panic しても inFlight が
// クリアされ、当該キーの後続タスクが実行可能なまま（永久ブロックしない）ことを確認する。
func TestAsyncCoalescerRecoversFromPanic(t *testing.T) {
	calls := make(chan int, 16)
	panics := make(chan struct{}, 4)
	var mu sync.Mutex
	n := 0
	c := newAsyncCoalescer(func(_ string) {
		mu.Lock()
		n++
		cur := n
		mu.Unlock()
		if cur == 1 {
			panic("boom")
		}
		calls <- cur
	})
	c.onPanic = func(_ string, _ any) { panics <- struct{}{} }

	c.trigger("g1")
	select {
	case <-panics:
	case <-time.After(2 * time.Second):
		t.Fatal("onPanic was not invoked")
	}

	// panic 後も inFlight がクリアされ、後続タスクが走ること（永久ブロックしない）。
	deadline := time.After(2 * time.Second)
	for {
		c.trigger("g1")
		select {
		case got := <-calls:
			if got >= 2 {
				return // 2回目以降が走った = inFlight はクリア済み
			}
		case <-time.After(20 * time.Millisecond):
		case <-deadline:
			t.Fatal("coalescer did not recover (inFlight stuck) after panic")
		}
	}
}

// TestAsyncCoalescerStopWaitsForInFlightAndBlocksNewTriggers は、stop が
// 実行中の run の完了を待ち、stop 後の trigger は run を起動しないことを確認する。
// バックアップ復元で DB を閉じる前に同期 goroutine を静止させる用途で必要。
func TestAsyncCoalescerStopWaitsForInFlightAndBlocksNewTriggers(t *testing.T) {
	released := make(chan struct{})
	started := make(chan struct{}, 1)
	var calls int
	var mu sync.Mutex
	c := newAsyncCoalescer(func(_ string) {
		mu.Lock()
		calls++
		mu.Unlock()
		select {
		case started <- struct{}{}:
		default:
		}
		<-released
	})

	c.trigger("g1")
	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("first run did not start")
	}

	// stop を別 goroutine で開始し、in-flight run の完了待ちでブロックすること。
	stopDone := make(chan struct{})
	go func() {
		c.stop()
		close(stopDone)
	}()

	select {
	case <-stopDone:
		t.Fatal("stop returned before the in-flight run finished")
	case <-time.After(100 * time.Millisecond):
	}

	// stop 中の trigger は run を起動してはいけない（既存 in-flight が終わっても
	// pending を消化しない）。
	c.trigger("g1")
	c.trigger("g2")

	close(released)

	select {
	case <-stopDone:
	case <-time.After(2 * time.Second):
		t.Fatal("stop did not return after in-flight run completed")
	}

	mu.Lock()
	got := calls
	mu.Unlock()
	if got != 1 {
		t.Fatalf("calls = %d, want 1 (stop 後の trigger は run を起動しない)", got)
	}

	// stop 後の trigger も何度呼んでも安全（no-op）。
	c.trigger("g1")
	time.Sleep(50 * time.Millisecond)
	mu.Lock()
	got = calls
	mu.Unlock()
	if got != 1 {
		t.Fatalf("post-stop trigger increased calls to %d", got)
	}
}

// @fileoverview ApiResult の基本動作を検証する。
package result

import "testing"

func TestOkResult(t *testing.T) {
	value := OkResult(42)
	if !value.Success {
		t.Fatalf("expected success to be true")
	}
	if value.Data != 42 {
		t.Fatalf("expected data to be 42")
	}
	if value.Error != nil {
		t.Fatalf("expected error to be nil")
	}
}

func TestErrorResult(t *testing.T) {
	value := ErrorResult[int]("failed", "detail")
	if value.Success {
		t.Fatalf("expected success to be false")
	}
	if value.Error == nil {
		t.Fatalf("expected error to be set")
	}
	if value.Error.Message != "failed" {
		t.Fatalf("unexpected error message")
	}
}

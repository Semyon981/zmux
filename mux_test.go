package zmux_test

import (
	"fmt"
	"testing"
)

func TestSample(t *testing.T) {
	for i := 0; i < 10; i++ {
		if x := fmt.Sprintf("%d", 45); x != "45" {
			t.Fatalf("Unexpected string: %s", x)
		}
	}
}

func BenchmarkSample(b *testing.B) {
	for i := 0; i < b.N; i++ {
		if x := fmt.Sprintf("%d", 46); x != "46" {
			b.Fatalf("Unexpected string: %s", x)
		}
	}
}

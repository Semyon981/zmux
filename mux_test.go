package zmux_test

import (
	"fmt"
	"testing"
)

func TestSample(t *testing.T) {
	for i := 0; i < 10; i++ {
		if x := fmt.Sprintf("%d", 42); x != "42" {
			t.Fatalf("Unexpected string: %s", x)
		}
	}
}

func BenchmarkSample(b *testing.B) {
	for i := 0; i < b.N; i++ {
		if x := fmt.Sprintf("%d", 42); x != "42" {
			b.Fatalf("Unexpected string: %s", x)
		}
	}
}

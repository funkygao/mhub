package broker

import (
	"testing"
)

func BenchmarkChannelClose(b *testing.B) {
	b.ReportAllocs()
	var c chan bool
	for i := 0; i < b.N; i++ {
		c = make(chan bool)
		close(c)
	}
}

package bench

import (
    "testing"
)

func BenchmarkChanRecv(b *testing.B) {
    c := make(chan bool, 100)
    go func() {
        for {
            select {
            case <-c:
            default:
            }
        }
    }()
    for i:=0; i<b.N; i++ {
        c <- true
    }
}

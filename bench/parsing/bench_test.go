package bench

import (
	"strings"
	"testing"
)

func BenchmarkSplit(b *testing.B) {
	b.ReportAllocs()
	s := `343434334,{"payload":{"330":{"uid":53,"march_id":330,"city_id":53,"opp_uid":0,"world_id":1,"type":"encamp","start_x":72,"start_y":64,"end_x":80,"end_y":78,"start_time":1412999095,"end_time":1412999111,"speed":1,"state":"marching","alliance_id":0}}`
	for i := 0; i < b.N; i++ {
		strings.SplitN(s, ",", 1)
	}
	b.SetBytes(int64(len(s)))
}

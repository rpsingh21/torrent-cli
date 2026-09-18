package tracker

import (
	"os"
	"testing"

	"github.com/rpsingh21/torrent-cli/internal/bencode"
)

var result *Response

func BenchmarkUnmarshalTrackerResponse(b *testing.B) {
	body, err := os.ReadFile("../../../testdata/metadata/httpbody.test.data")
	if err != nil {
		b.Fatal(err)
	}

	decodeData, err := bencode.NewDecoder(body).Decode()
	if err != nil {
		b.Fatal(err)
	}

	b.ReportAllocs()
	b.SetBytes(int64(len(body)))

	for b.Loop() {
		unData, err := UnmarshalTrackerResponse(decodeData)
		if err != nil {
			b.Fatal(err)
		}
		result = unData
	}
}

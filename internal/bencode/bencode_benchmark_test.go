package bencode

import (
	"os"
	"path/filepath"
	"testing"
)

func loadTorrent(b *testing.B, filename string) []byte {
	b.Helper()

	data, err := os.ReadFile(
		filepath.Join("../../testdata/torrents", filename),
	)
	if err != nil {
		b.Fatal(err)
	}

	return data
}

// Prevent compiler optimizations from removing benchmark results.
var benchmarkResult any

func benchmarkDecode(b *testing.B, filename string) {
	data := loadTorrent(b, filename)

	b.ReportAllocs()
	b.SetBytes(int64(len(data)))

	for b.Loop() {
		result, err := NewDecoder(data).Decode()
		if err != nil {
			b.Fatal(err)
		}

		benchmarkResult = result
	}
}

func benchmarkEncode(b *testing.B, filename string) {
	data := loadTorrent(b, filename)

	decoded, err := NewDecoder(data).Decode()
	if err != nil {
		b.Fatal(err)
	}

	b.ReportAllocs()
	b.SetBytes(int64(len(data)))

	for b.Loop() {
		result, err := Encode(decoded)
		if err != nil {
			b.Fatal(err)
		}

		benchmarkResult = result
	}
}

func benchmarkRoundTrip(b *testing.B, filename string) {
	data := loadTorrent(b, filename)

	b.ReportAllocs()
	b.SetBytes(int64(len(data)))

	for b.Loop() {
		decoded, err := NewDecoder(data).Decode()
		if err != nil {
			b.Fatal(err)
		}

		result, err := Encode(decoded)
		if err != nil {
			b.Fatal(err)
		}

		benchmarkResult = result
	}
}

// --------------------
// Decoder benchmarks
// --------------------

func BenchmarkDecodeMultiFile(b *testing.B) {
	benchmarkDecode(b, "test.torrent")
}

func BenchmarkDecodeSingleFile(b *testing.B) {
	benchmarkDecode(b, "singlefile.torrent")
}

func BenchmarkDecodeArchlinux(b *testing.B) {
	benchmarkDecode(b, "archlinux-2026.09.01.torrent")
}

func BenchmarkDecodeUbuntu(b *testing.B) {
	benchmarkDecode(b, "ubuntu-26.04.1.torrent")
}

// --------------------
// Encoder benchmarks
// --------------------

func BenchmarkEncodeMultiFile(b *testing.B) {
	benchmarkEncode(b, "test.torrent")
}

func BenchmarkEncodeSingleFile(b *testing.B) {
	benchmarkEncode(b, "singlefile.torrent")
}

func BenchmarkEncodeArchlinux(b *testing.B) {
	benchmarkEncode(b, "archlinux-2026.09.01.torrent")
}

func BenchmarkEncodeUbuntu(b *testing.B) {
	benchmarkEncode(b, "ubuntu-26.04.1.torrent")
}

// --------------------
// Round-trip benchmarks
// --------------------

func BenchmarkRoundTripMultiFile(b *testing.B) {
	benchmarkRoundTrip(b, "test.torrent")
}

func BenchmarkRoundTripSingleFile(b *testing.B) {
	benchmarkRoundTrip(b, "singlefile.torrent")
}

func BenchmarkRoundTripArchlinux(b *testing.B) {
	benchmarkRoundTrip(b, "archlinux-2026.09.01.torrent")
}

func BenchmarkRoundTripUbuntu(b *testing.B) {
	benchmarkRoundTrip(b, "ubuntu-26.04.1.torrent")
}

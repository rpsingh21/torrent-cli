package bitmaskreservoir

import (
	"fmt"
	"testing"
)

func TestBlockReservoirPositive(t *testing.T) {
	br := NewBitmaskReservoir(150)

	for i := range 150 {
		id, err := br.Reserve()
		if err != nil {
			t.Fatalf("unexpected error found: %v", err)
		}

		if id != i {
			t.Fatalf("id should be %v but reserved %v", i, id)
		}

		if isReserved := br.IsReserved(i); !isReserved {
			t.Fatalf("Id %v should reserved", i)
		}
	}

	if _, err := br.Reserve(); err == nil || err != ErrNoBlocksAvailable {
		t.Fatalf("expted error ErrNoBlocksAvailable but err :%v", err)
	}

	for _, id := range []int{23, 50, 63, 64, 123, 124, 149} {

		if err := br.Unreserve(id); err != nil {
			t.Fatalf("unexpected error found: %v", err)
		}

		if isReserved := br.IsReserved(id); isReserved {
			t.Fatalf("Id %v should unreserve", id)
		}

		if idx, err := br.Reserve(); err != nil || idx != id {
			t.Fatalf("unexpected result found: value: %v, err: %v", id, err)
		}
	}
}

func TestBlockReservoirNegative(t *testing.T) {
	br := NewBitmaskReservoir(150)

	if val := br.IsReserved(-1); val {
		t.Fatalf("expected false found: %v", val)
	}

	if val := br.IsReserved(150); val {
		t.Fatalf("expected false found: %v", val)
	}

	if err := br.Unreserve(-1); err == nil {
		t.Fatalf("expected err, found: %v", err)
	}

	if err := br.Unreserve(-1); err == nil {
		t.Fatalf("expected err, found: %v", err)
	}

	if br := NewBitmaskReservoir(0); br != nil {
		t.Fatalf("expected nil, found: %v", br)
	}
}

func BenchmarkBitmaskReservoir(b *testing.B) {
	for _, count := range []int{1, 2, 4, 8, 16, 32, 64, 128, 256, 512, 1024} {
		b.Run(fmt.Sprintf("BC_%d", count), func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				res := NewBitmaskReservoir(count)
				for range count {
					_, _ = res.Reserve()
				}
			}
		})
	}
}

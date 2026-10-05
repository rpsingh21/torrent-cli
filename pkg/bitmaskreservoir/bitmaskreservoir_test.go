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

	if _, err := br.Reserve(); err == nil || err != ErrNoBitAvailable {
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

func TestCanReserve(t *testing.T) {
	br := NewBitmaskReservoir(1088)
	for range 1050 {
		br.Reserve()
	}

	if err := br.Unreserve(64); err != nil {
		t.Fatalf("Unexpted error, %v", err)
	}

	if !br.CanReserve() {
		t.Fatal("expcted true, return false")
	}

	if val, err := br.Reserve(); err != nil || val != 64 {
		t.Fatalf("Unexpted result val %v, err %v", val, err)
	}

	if !br.CanReserve() {
		t.Fatal("expcted true, return false")
	}

	if val, err := br.Reserve(); err != nil || val != 1050 {
		t.Fatalf("Unexpted result val %v, err %v", val, err)
	}

	if val, err := br.Reserve(); err != nil || val != 1051 {
		t.Fatalf("Unexpted result val %v, err %v", val, err)
	}

	if !br.CanReserve() {
		t.Fatal("expcted true, return false")
	}
}

func TestCountReserved(t *testing.T) {
	br := NewBitmaskReservoir(150)
	for range 150 {
		br.Reserve()
	}

	if count := br.CountReserved(); count != 150 {
		t.Fatalf("expected count 150, got %v", count)
	}

	for i, id := range []int{23, 50, 63, 64, 123, 124, 149} {
		br.Unreserve(id)

		if count := br.CountReserved(); count != (149 - i) {
			t.Fatalf("expected count %v, got %v", (149 - i), count)
		}
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

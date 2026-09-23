package datastructure

import (
	"testing"
)

func TestDqueue_PopRight(t *testing.T) {
	tests := []struct {
		name       string
		setup      []int
		wantValue  int
		wantOK     bool
		wantLength int
	}{
		{
			name:       "empty deque",
			setup:      []int{},
			wantValue:  0,
			wantOK:     false,
			wantLength: 0,
		},
		{
			name:       "single element",
			setup:      []int{10},
			wantValue:  10,
			wantOK:     true,
			wantLength: 0,
		},
		{
			name:       "multiple elements",
			setup:      []int{10, 20, 30},
			wantValue:  30,
			wantOK:     true,
			wantLength: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q := NewDqueue[int]()

			for _, value := range tt.setup {
				q.AddEnd(value)
			}

			got, ok := q.PopRight()

			if got != tt.wantValue {
				t.Fatalf("value = %v, want %v", got, tt.wantValue)
			}

			if ok != tt.wantOK {
				t.Fatalf("ok = %v, want %v", ok, tt.wantOK)
			}

			if q.Length != tt.wantLength {
				t.Fatalf("Length = %d, want %d", q.Length, tt.wantLength)
			}
		})
	}
}

func TestDqueue_PopRight_Pointers(t *testing.T) {
	q := NewDqueue[int]()

	q.Add(10)
	q.AddEnd(20)
	q.AddEnd(30)

	got, ok := q.PopRight()

	if !ok {
		t.Fatal("expected PopRight to succeed")
	}

	if got != 30 {
		t.Fatalf("got %d, want 30", got)
	}

	if q.tail == nil {
		t.Fatal("tail should not be nil")
	}

	if q.tail.Data != 20 {
		t.Fatalf("tail.Data = %d, want 20", q.tail.Data)
	}

	if q.tail.right != nil {
		t.Fatal("tail.right should be nil")
	}

	// Head should still be 10.
	if q.head == nil || q.head.Data != 10 {
		t.Fatal("head should be 10")
	}

	// Verify bidirectional relationship.
	if q.head.right != q.tail {
		t.Fatal("head.right should point to tail")
	}

	if q.tail.left != q.head {
		t.Fatal("tail.left should point to head")
	}
}

func TestDqueue_PopRight_SingleElement(t *testing.T) {
	q := NewDqueue[int]()

	q.AddEnd(100)

	got, ok := q.PopRight()

	if !ok {
		t.Fatal("expected ok=true")
	}

	if got != 100 {
		t.Fatalf("got %d, want 100", got)
	}

	if q.Length != 0 {
		t.Fatalf("Length = %d, want 0", q.Length)
	}

	if q.head != nil {
		t.Fatal("head should be nil")
	}

	if q.tail != nil {
		t.Fatal("tail should be nil")
	}
}

func TestDqueue_AddFront_PopLeft(t *testing.T) {
	q := NewDqueue[int]()

	// Add to front:
	// 30
	q.AddFront(30)

	// 20 <-> 30
	q.AddFront(20)

	// 10 <-> 20 <-> 30
	q.AddFront(10)

	if q.Length != 3 {
		t.Fatalf("Length = %d, want 3", q.Length)
	}

	if q.head.Data != 10 {
		t.Fatalf("head.Data = %d, want 10", q.head.Data)
	}

	if q.tail.Data != 30 {
		t.Fatalf("tail.Data = %d, want 30", q.tail.Data)
	}

	// Pop from left.
	got, ok := q.Pop()

	if !ok {
		t.Fatal("PopRight() returned ok=false, want true")
	}

	if got != 10 {
		t.Fatalf("PopRight() = %d, want 30", got)
	}

	if q.Length != 2 {
		t.Fatalf("Length = %d, want 2", q.Length)
	}

	// Remaining:
	// 20 <-> 30
	if q.head.Data != 20 {
		t.Fatalf("head.Data = %d, want 10", q.head.Data)
	}

	if q.tail.Data != 30 {
		t.Fatalf("tail.Data = %d, want 20", q.tail.Data)
	}

	// Check bidirectional links.
	if q.head.right != q.tail {
		t.Fatal("head.right does not point to tail")
	}

	if q.tail.left != q.head {
		t.Fatal("tail.left does not point to head")
	}

	// New tail must not point right.
	if q.tail.right != nil {
		t.Fatal("tail.right should be nil")
	}

	// Head must not point left.
	if q.head.left != nil {
		t.Fatal("head.left should be nil")
	}
}

func TestDqueue_PopLeft_SingleElement(t *testing.T) {
	q := NewDqueue[int]()

	q.AddEnd(100)

	got, ok := q.Pop()

	if !ok {
		t.Fatal("expected ok=true")
	}

	if got != 100 {
		t.Fatalf("got %d, want 100", got)
	}

	if q.Length != 0 {
		t.Fatalf("Length = %d, want 0", q.Length)
	}

	if q.head != nil {
		t.Fatal("head should be nil")
	}

	if q.tail != nil {
		t.Fatal("tail should be nil")
	}
}

func TestDqueue_PopLeft_Empty(t *testing.T) {
	q := NewDqueue[int]()

	got, ok := q.Pop()

	if ok {
		t.Fatal("PopRight() returned ok=true, want false")
	}

	if got != 0 {
		t.Fatalf("got %d, want zero value 0", got)
	}
}

func BenchmarkDqueue_PopRight(b *testing.B) {
	q := NewDqueue[int]()

	for b.Loop() {
		q.AddEnd(1)
		q.PopRight()
	}
}

func BenchmarkDqueue_PopRight_add3(b *testing.B) {
	for b.Loop() {
		q := NewDqueue[int]()

		q.Add(1)
		q.AddEnd(2)
		q.AddEnd(3)

		q.PopRight()
	}
}

func BenchmarkDqueue_Pop_Large(b *testing.B) {
	for b.Loop() {
		q := NewDqueue[int]()

		for j := range 100000 {
			q.AddEnd(j)
		}

		q.Pop()
	}
}

func BenchmarkDqueue_PopRight_Large(b *testing.B) {
	for b.Loop() {
		q := NewDqueue[int]()

		for j := range 100000 {
			q.AddEnd(j)
		}

		q.PopRight()
	}
}

func BenchmarkDqueue_PoolReuse(b *testing.B) {
	q := NewDqueue[int]()

	b.ReportAllocs()

	for b.Loop() {
		for i := range 100_000 {
			q.AddEnd(i)
		}

		for range 100_000 {
			q.PopRight()
		}
	}
}

func BenchmarkSimplequeueWithoutPreAllocation_PoolReuse(b *testing.B) {
	q := NewQueue[int]()
	// q := make(Queue[int], 0, 100_000)

	b.ReportAllocs()

	for b.Loop() {
		for i := range 100_000 {
			q.Push(i)
		}

		for range 100_000 {
			q.Pop()
		}
	}
}

func BenchmarkSimplequeuePreAllocation_PoolReuse(b *testing.B) {
	q := make(Queue[int], 10)

	for b.Loop() {
		for i := range 100_000 {
			q = append(q, i)
		}

		for range 100_000 {
			q.Pop()
		}
	}
}

func BenchmarkQueue(b *testing.B) {
	const N = 100_000

	q := make(Queue[int], 0, N)

	b.ReportAllocs()

	for b.Loop() {
		for i := 0; i < N; i++ {
			q = append(q, i)
		}

		for i := 0; i < N; i++ {
			_, _ = q.Pop()
		}

		q = q[:0]
	}
}

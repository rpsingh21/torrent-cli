package datastructure

type Queue[T any] []T

func NewQueue[T any]() *Queue[T] {
	return &Queue[T]{}
}

func (q *Queue[T]) Push(data T) {
	*q = append(*q, data)
}

func (q *Queue[T]) Peek() (T, bool) {
	if len(*q) == 0 {
		var zero T
		return zero, false
	}
	return (*q)[0], true
}

func (q *Queue[T]) Pop() (T, bool) {
	if len(*q) == 0 {
		var zero T
		return zero, false
	}

	data := (*q)[0]
	*q = (*q)[1:]

	return data, true
}

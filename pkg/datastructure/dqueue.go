package datastructure

type DQueueNode[T any] struct {
	left  *DQueueNode[T]
	right *DQueueNode[T]
	Data  T
}

type Dqueue[T any] struct {
	Length int
	head   *DQueueNode[T]
	tail   *DQueueNode[T]
}

func NewDQueueNode[T any](data T) *DQueueNode[T] {
	return &DQueueNode[T]{
		Data: data,
	}
}

func NewDqueue[T any]() *Dqueue[T] {
	return &Dqueue[T]{}
}

func (dque *Dqueue[T]) Add(data T) {
	dque.AddEnd(data)
}

func (dque *Dqueue[T]) AddEnd(data T) {
	node := NewDQueueNode(data)

	if dque.Length == 0 {
		dque.head = node
		dque.tail = node
	} else {
		node.left = dque.tail
		dque.tail.right = node
		dque.tail = node
	}

	dque.Length++
}

func (dque *Dqueue[T]) AddFront(data T) {
	node := NewDQueueNode(data)

	if dque.Length == 0 {
		dque.head = node
		dque.tail = node
	} else {
		node.right = dque.head
		dque.head.left = node
		dque.head = node
	}
	dque.Length++
}

func (dque *Dqueue[T]) Pop() (T, bool) {
	return dque.PopLeft()
}

func (dque *Dqueue[T]) PopLeft() (T, bool) {
	if dque.Length == 0 {
		var zero T
		return zero, false
	}

	node := dque.head
	dque.head = node.right

	if dque.head != nil {
		dque.head.left = nil
	} else {
		dque.tail = nil
	}

	dque.Length--

	return node.Data, true
}

func (dque *Dqueue[T]) PopRight() (T, bool) {
	if dque.Length == 0 {
		var zero T
		return zero, false
	}

	node := dque.tail
	dque.tail = node.left

	if dque.tail != nil {
		dque.tail.right = nil
	} else {
		dque.head = nil
	}

	dque.Length--
	return node.Data, true
}

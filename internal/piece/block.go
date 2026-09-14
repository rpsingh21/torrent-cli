package piece

type Block struct {
	Piece     uint32
	Offset    uint32
	Length    uint32
	data      []byte
	requested bool
	completed bool
}

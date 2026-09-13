package piece

type Block struct {
	Piece     int
	offset    int
	Length    int
	data      []byte
	requested bool
	completed bool
}

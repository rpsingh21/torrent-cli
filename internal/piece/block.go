package piece

type Block struct {
	Piece     int
	Offset    int
	Length    int
	Data      []byte
	Requested bool
	Completed bool
	startedAt int64
}

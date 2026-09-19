package piece

import "time"

type Block struct {
	Piece     uint32
	Offset    uint32
	Length    uint32
	Data      []byte
	Requested bool
	Completed bool
	startedAt time.Time
}

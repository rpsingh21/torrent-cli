package piece

import "time"

type Block struct {
	Piece       int
	Offset      int
	Length      int
	Data        []byte
	Requested   bool
	RequestedBy string
	Completed   bool
	startedAt   time.Time
}

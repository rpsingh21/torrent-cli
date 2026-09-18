package storage

type Storage interface {
	ReadAt(p []byte, offset int64) (int, error)

	WriteAt(p []byte, offset int64) (int, error)

	ReadPiece(index int, dst []byte) error

	WritePiece(index int, src []byte) error

	VerifyPiece(index int) (bool, error)
	PieceComplete(index int) bool

	Close() error
}

package storage

type Storage interface {
	ReadAt(p []byte, offset int64) (int, error)

	WriteAt(p []byte, offset int64) (int, error)

	Sync() error

	Close() error
}

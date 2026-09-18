package storage

import (
	"io"
	"log"
	"os"
	"path/filepath"

	"github.com/rpsingh21/torrent-cli/internal/torrent"
)

type FileStorage struct {
	metaInfo *torrent.MetaInfo
	offsets  []int64
	files    []*os.File
}

func NewFileStorage(metaInfo *torrent.MetaInfo, baseDir string) (*FileStorage, error) {

	n := len(metaInfo.Files)
	offsets, files := make([]int64, n+1), make([]*os.File, n)
	offsets[0] = 0

	if err := os.RemoveAll(baseDir); err != nil {
		log.Printf("Getting error while removing folder : %v", err)
	}

	for i, tf := range metaInfo.Files {
		filePath := filepath.Join(baseDir, tf.Path)

		if err := os.MkdirAll(filepath.Dir(filePath), 0775); err != nil {
			return nil, err
		}

		file, err := os.OpenFile(
			filePath,
			os.O_WRONLY|os.O_CREATE|os.O_TRUNC,
			0644,
		)
		if err != nil {
			return nil, err
		}

		files[i] = file
		offsets[i+1] = offsets[i] + tf.Length
	}

	return &FileStorage{
		metaInfo: metaInfo,
		offsets:  offsets,
		files:    files,
	}, nil

}

// func (f *FileStorage) ReadAt(p []byte, offset int64) (int, error) {
// 	return 0, error.Error("Not implemented")
// }

// func (f *FileStorage) WriteAt(p []byte, offset int64) (int, error) {

// }

// ReadPiece(index int, dst []byte) error

func (fs *FileStorage) WritePiece(index int, src []byte) error {
	pieceStart := int64(index) * fs.metaInfo.PieceLength
	pieceEnd := pieceStart + int64(len(src))

	fileIndex := fs.getFileIndex(pieceStart)

	srcOffset := int64(0)

	for pieceStart < pieceEnd {
		fileStart := fs.offsets[fileIndex]
		fileEnd := fs.offsets[fileIndex+1]

		// Where inside the current file do we start?
		fileOffset := pieceStart - fileStart

		// Number of bytes we can write to this file.
		n := min(pieceEnd-pieceStart, fileEnd-pieceStart)

		written, err := fs.files[fileIndex].WriteAt(
			src[srcOffset:srcOffset+n],
			fileOffset,
		)
		if err != nil {
			return err
		}

		if int64(written) != n {
			return io.ErrShortWrite
		}

		pieceStart += n
		srcOffset += n
		fileIndex++
	}

	return nil
}

// VerifyPiece(index int) (bool, error)

// PieceComplete(index int) bool

func (fs *FileStorage) Close() {
	for _, file := range fs.files {
		file.Close()
	}
}

// It will take piece index and return file index.
// So data write on corrent file
func (fs *FileStorage) getFileIndex(target int64) int {
	left := 0
	right := len(fs.offsets) - 1

	for left < right {
		mid := (left + right) >> 1

		if fs.offsets[mid+1] <= target {
			left = mid + 1
		} else {
			right = mid
		}
	}

	return left
}

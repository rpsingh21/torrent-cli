package storage

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"

	"github.com/rpsingh21/torrent-cli/internal/torrent"
)

type FileStorage struct {
	metaInfo *torrent.MetaInfo
	offsets  []int64
	files    []*os.File
}

func NewFileStorage(metaInfo *torrent.MetaInfo, baseDir string) (*FileStorage, error) {

	if len(metaInfo.Files) == 0 {
		return nil, fmt.Errorf("Torrent contains no files")
	}

	n := len(metaInfo.Files)
	offsets, files := make([]int64, n+1), make([]*os.File, n)
	offsets[0] = 0

	fileStorage := &FileStorage{
		metaInfo: metaInfo,
		offsets:  offsets,
		files:    files,
	}

	cleanup := func(err error) (*FileStorage, error) {
		_ = fileStorage.Close()
		return nil, err
	}

	for i, tf := range metaInfo.Files {
		if tf.Length < 0 {
			return cleanup(fmt.Errorf("negative file length for %q", tf.Path))
		}

		filePath := filepath.Join(baseDir, tf.Path)
		if err := os.MkdirAll(filepath.Dir(filePath), 0775); err != nil {
			return cleanup(fmt.Errorf("Create directory for %q: %w", tf.Path, err))
		}

		file, err := os.OpenFile(
			filePath,
			os.O_RDWR|os.O_CREATE,
			0644,
		)
		if err != nil {
			return cleanup(err)
		}

		files[i] = file
		offsets[i+1] = offsets[i] + tf.Length
	}

	return fileStorage, nil

}

func (f *FileStorage) ReadAt(buf []byte, offset int64) (int, error) {
	if offset < 0 {
		return 0, fmt.Errorf("negative offset: %d", offset)
	}

	if len(buf) == 0 {
		return 0, nil
	}

	totalSize := f.metaInfo.TotalSize
	if offset >= totalSize {
		return 0, io.EOF
	}

	totalRead := 0

	for totalRead < len(buf) {
		currentOffset := offset + int64(totalRead)
		if currentOffset >= totalSize {
			return totalRead, io.EOF
		}

		fileIndex := f.getFileIndex(currentOffset)
		if fileIndex < 0 || fileIndex >= len(f.files) {
			return totalRead, io.EOF
		}

		fileStart := f.offsets[fileIndex]
		fileEnd := f.offsets[fileIndex+1]

		fileOffset := currentOffset - fileStart
		availableInFile := fileEnd - currentOffset
		remainingToRead := int64(len(buf) - totalRead)

		toRead := min(availableInFile, remainingToRead)

		n, err := f.files[fileIndex].ReadAt(buf[totalRead:totalRead+int(toRead)], fileOffset)
		totalRead += n

		if err != nil && !errors.Is(err, io.EOF) {
			return totalRead, fmt.Errorf("read error on file %d at offset %d: %w", fileIndex, fileOffset, err)
		}

		// If a file yielded fewer bytes than expected, we hit premature EOF
		if int64(n) < toRead {
			return totalRead, io.EOF
		}
	}

	return totalRead, nil
}

func (f *FileStorage) ReadPiece(index int, buf []byte) error {
	offset := int64(index) * f.metaInfo.PieceLength

	if _, err := f.ReadAt(buf, offset); err != nil {
		return err
	}

	return nil
}

// Todo: Will imp
func (f *FileStorage) WriteAt(p []byte, offset int64) (int, error) {
	return 0, fmt.Errorf("Not implemented")
}

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

func (fs *FileStorage) Close() error {
	var firstErr error
	for _, file := range fs.files {
		if err := file.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}

	return firstErr
}

func (f *FileStorage) getFileIndex(offset int64) int {
	// f.offsets should have len(files) + 1 entries: [0, size0, size0+size1, ...]
	// We look for the largest index i such that f.offsets[i] <= offset
	idx := sort.Search(len(f.offsets), func(i int) bool {
		return f.offsets[i] > offset
	}) - 1

	return idx
}

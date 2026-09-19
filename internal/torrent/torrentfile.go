package torrent

import (
	"crypto/rand"
	"crypto/sha1"
	"fmt"
	"os"
	"strings"

	"github.com/rpsingh21/torrent-cli/internal/bencode"
)

func NewTorrentDetailFromFile(filePath string) (*MetaInfo, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("Read torrent file: %w", err)
	}

	decoded, err := bencode.NewDecoder(data).Decode()
	if err != nil {
		return nil, err
	}

	root, ok := decoded.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("Invalid torrent: root is not a dictionary")
	}

	torrent := &MetaInfo{
		AppId: calculate_peer_id(),
	}

	if announce, ok := root["announce"].([]byte); ok {
		torrent.Announce = string(announce)
	}

	info, ok := root["info"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("Invalid torrent: missing info dictionary")
	}

	infoByte, err := bencode.Encode(info)
	if err != nil {
		return nil, err
	}
	torrent.InfoHash = sha1.Sum(infoByte)

	if name, ok := info["name"].([]byte); ok {
		torrent.Name = string(name)
	}

	if length, ok := info["length"].(int64); ok {
		torrent.Length = length
	}

	if pieceLength, ok := info["piece length"].(int64); ok {
		torrent.PieceLength = pieceLength
	}

	if pieces, ok := info["pieces"].([]byte); ok {
		hashes, err := splitPieceHashes(pieces)
		if err != nil {
			return nil, err
		}
		torrent.PieceHashes = hashes
	}

	torrent.Files, torrent.TotalSize = parseFiles(info, torrent.Length, torrent.Name)

	return torrent, nil
}

func parseFiles(info map[string]any, length int64, name string) ([]TFile, int64) {
	files, ok := info["files"].([]any)
	var totalSize int64
	if !ok {
		return []TFile{
			{
				Length: length,
				Path:   name,
			},
		}, length
	}

	result := make([]TFile, 0, len(files))

	for _, raw := range files {
		file, ok := raw.(map[string]any)
		if !ok {
			continue
		}

		fileLength, ok := file["length"].(int64)
		if !ok {
			continue
		}

		path, ok := file["path"].([]any)
		if !ok {
			continue
		}

		result = append(result, TFile{
			Length: fileLength,
			Path:   bytesPathToString(path),
		})
		totalSize += fileLength
	}

	return result, totalSize
}

func bytesPathToString(path []any) string {
	if len(path) == 0 {
		return ""
	}

	var builder strings.Builder

	for _, part := range path {
		value, ok := part.([]byte)
		if !ok {
			continue
		}

		if builder.Len() > 0 {
			builder.WriteByte('/')
		}

		builder.Write(value)
	}

	return builder.String()
}

func splitPieceHashes(pieces []byte) ([][20]byte, error) {
	const pieceHashLen = 20
	if len(pieces)%pieceHashLen != 0 {
		return nil, fmt.Errorf(
			"Malformed pieces: length %d is not divisible by %d",
			len(pieces),
			pieceHashLen,
		)
	}

	hashes := make([][20]byte, len(pieces)/pieceHashLen)

	for i := range hashes {
		start := i * pieceHashLen
		copy(hashes[i][:], pieces[start:start+pieceHashLen])
	}

	return hashes, nil
}

func calculate_peer_id() [20]byte {
	var peerID [20]byte
	rand.Read(peerID[:])
	return peerID
}

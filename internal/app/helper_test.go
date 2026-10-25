package app

import (
	"log"
	"testing"
)

func TestDowloaddata(t *testing.T) {
	URL := "magnet:?xt=urn:btih:82075af82e83b99e9a4c6ef17e564947efe65a66&tr=https%3A%2F%2Facademictorrents.com%2Fannounce.php&tr=udp%3A%2F%2Ftracker.coppersurfer.tk%3A6969&tr=udp%3A%2F%2Ftracker.opentrackr.org%3A1337%2Fannounce"
	metainfo, err := getMetaInfo(t.Context(), URL)

	if err != nil {
		t.Fatalf("Failed metadata upload: %v", err)
	}

	log.Printf("%v", metainfo.PieceLength)
}

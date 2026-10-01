package torrent

import (
	"log"
	"testing"
)

func TestMagnetURL(t *testing.T) {
	URL := "magnet:?xt=urn:btih:5AC862A41FBF4643E1FA33F57098357176C91CEC&dn=Mr.%20Bean%20-%20The%20Complete%20Collection%20(1990-2007)&tr=udp%3A%2F%2Ftracker.opentrackr.org%3A1337&tr=udp%3A%2F%2Fopen.stealth.si%3A80%2Fannounce&tr=udp%3A%2F%2Ftracker.torrent.eu.org%3A451%2Fannounce&tr=udp%3A%2F%2Ftracker.bittor.pw%3A1337%2Fannounce&tr=udp%3A%2F%2Fpublic.popcorn-tracker.org%3A6969%2Fannounce&tr=udp%3A%2F%2Ftracker.dler.org%3A6969%2Fannounce&tr=udp%3A%2F%2Fexodus.desync.com%3A6969&tr=udp%3A%2F%2Fopen.demonii.com%3A1337%2Fannounce&tr=udp%3A%2F%2Fglotorrents.pw%3A6969%2Fannounce&tr=udp%3A%2F%2Ftracker.coppersurfer.tk%3A6969&tr=udp%3A%2F%2Ftorrent.gresille.org%3A80%2Fannounce&tr=udp%3A%2F%2Fp4p.arenabg.com%3A1337&tr=udp%3A%2F%2Ftracker.internetwarriors.net%3A1337"

	metaInfo, err := MetaInfoFromMagnetURL(URL)
	if err != nil {
		t.Fatalf("Faild to get metainfo from magnet URL %v", err)
	}

	log.Printf("%+v", metaInfo.Announce)
}

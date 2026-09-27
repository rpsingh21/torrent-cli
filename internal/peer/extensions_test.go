package peer

import (
	"testing"

	"github.com/rpsingh21/torrent-cli/internal/torrent"
)

func TestExtendedHandshake(t *testing.T) {
	URL := "magnet:?xt=urn:btih:5AC862A41FBF4643E1FA33F57098357176C91CEC&dn=Mr.%20Bean%20-%20The%20Complete%20Collection%20(1990-2007)&tr=udp%3A%2F%2Ftracker.opentrackr.org%3A1337&tr=udp%3A%2F%2Fopen.stealth.si%3A80%2Fannounce&tr=udp%3A%2F%2Ftracker.torrent.eu.org%3A451%2Fannounce&tr=udp%3A%2F%2Ftracker.bittor.pw%3A1337%2Fannounce&tr=udp%3A%2F%2Fpublic.popcorn-tracker.org%3A6969%2Fannounce&tr=udp%3A%2F%2Ftracker.dler.org%3A6969%2Fannounce&tr=udp%3A%2F%2Fexodus.desync.com%3A6969&tr=udp%3A%2F%2Fopen.demonii.com%3A1337%2Fannounce&tr=udp%3A%2F%2Fglotorrents.pw%3A6969%2Fannounce&tr=udp%3A%2F%2Ftracker.coppersurfer.tk%3A6969&tr=udp%3A%2F%2Ftorrent.gresille.org%3A80%2Fannounce&tr=udp%3A%2F%2Fp4p.arenabg.com%3A1337&tr=udp%3A%2F%2Ftracker.internetwarriors.net%3A1337"

	// URL := "magnet:?xt=urn:btih:7d6adf33cd9976877fdc4b487607c3bdaf733241&tr=udp%3A%2F%2Ftracker.coppersurfer.tk%3A6969&tr=udp%3A%2F%2Ftracker.opentrackr.org%3A1337%2Fannounce"
	metaInfo, err := torrent.MetaInfoFromMagnetURL(URL)
	if err != nil {
		t.Fatal(err)
	}

	peers := []struct {
		ip   string
		port uint16
	}{
		{"82.15.154.129", 55175},
		{"178.162.159.33", 49114},
		// {"178.162.159.38", 50194},
		// {"185.148.1.103", 53740},
		// {"45.87.251.170", 39360},
		// {"101.190.48.90", 51413},
	}
	for _, p := range peers {
		t.Run(p.ip, func(t *testing.T) {
			peer := NewPeer(p.ip, p.ip, p.port, metaInfo, nil)

			data, err := peer.DownloadMetadata(t.Context(), metaInfo)
			if err != nil {
				t.Logf("peer %s failed metadata download: %v", p.ip, err)
				return
			}

			t.Logf(
				"peer %s successfully returned %d bytes",
				p.ip,
				len(data),
			)
		})
	}
}

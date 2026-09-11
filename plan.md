v0.1
│
├── bencode
├── .torrent parsing
└── info hash

v0.2
│
├── tracker
├── peer discovery
└── handshake

v0.3
│
├── piece downloading
├── piece verification
└── disk storage

v0.4
│
├── concurrent peers
├── choking/unchoking
├── piece selection
└── resume

v0.5
│
├── magnet links
├── DHT
└── UDP tracker

v0.6
│
├── TUI
├── multiple torrents
└── persistent state

v1.0
│
├── robust error handling
├── graceful shutdown
├── tests
├── benchmarks
├── observability/logging
├── configuration
└── documentation




pytorrent/
│
├── cmd/
│   └── pytorrent/
│       └── main.go
│
├── internal/
│
│   ├── app/
│   │   ├── app.go
│   │   └── lifecycle.go
│   │
│   ├── torrent/
│   │   ├── torrent.go
│   │   ├── metadata.go
│   │   ├── metainfo.go
│   │   ├── magnet.go
│   │   ├── v1.go
│   │   ├── v2.go
│   │   └── hybrid.go
│   │
│   ├── bencode/
│   │   ├── decoder.go
│   │   ├── encoder.go
│   │   ├── reader.go
│   │   └── writer.go
│   │
│   ├── peer/
│   │   ├── peer.go
│   │   ├── connection.go
│   │   ├── handshake.go
│   │   ├── message.go
│   │   ├── protocol.go
│   │   ├── extensions.go
│   │   ├── request.go
│   │   └── pool.go
│   │
│   ├── discovery/
│   │   ├── discovery.go
│   │   ├── queue.go
│   │   ├── tracker/
│   │   │   ├── tracker.go
│   │   │   ├── http.go
│   │   │   ├── udp.go
│   │   │   └── response.go
│   │   │
│   │   ├── dht/
│   │   │   ├── dht.go
│   │   │   ├── krpc.go
│   │   │   ├── message.go
│   │   │   ├── routing.go
│   │   │   ├── bucket.go
│   │   │   ├── node.go
│   │   │   ├── transaction.go
│   │   │   └── token.go
│   │   │
│   │   └── pex/
│   │       ├── pex.go
│   │       └── message.go
│   │
│   ├── metadata/
│   │   ├── metadata.go
│   │   ├── ut_metadata.go
│   │   └── piece.go
│   │
│   ├── piece/
│   │   ├── manager.go
│   │   ├── picker.go
│   │   ├── availability.go
│   │   ├── piece.go
│   │   ├── block.go
│   │   ├── endgame.go
│   │   └── verify.go
│   │
│   ├── download/
│   │   ├── engine.go
│   │   ├── scheduler.go
│   │   ├── pipeline.go
│   │   └── rate.go
│   │
│   ├── upload/
│   │   ├── engine.go
│   │   ├── choking.go
│   │   ├── slots.go
│   │   └── rate.go
│   │
│   ├── storage/
│   │   ├── storage.go
│   │   ├── file.go
│   │   ├── mmap.go
│   │   ├── sparse.go
│   │   ├── cache.go
│   │   └── resume.go
│   │
│   ├── session/
│   │   ├── session.go
│   │   ├── state.go
│   │   └── persistence.go
│   │
│   ├── config/
│   │   ├── config.go
│   │   └── defaults.go
│   │
│   ├── metrics/
│   │   ├── metrics.go
│   │   └── counters.go
│   │
│   └── tui/
│       ├── tui.go
│       ├── screen.go
│       ├── input.go
│       ├── render.go
│       └── pages/
│           ├── torrents.go
│           ├── peers.go
│           ├── files.go
│           └── trackers.go
│
├── pkg/
│   └── protocol/
│       └── types.go
│
├── configs/
│   └── config.example.toml
│
├── data/
│
├── testdata/
│   ├── torrents/
│   ├── magnets/
│   └── protocol/
│
├── docs/
│   ├── architecture.md
│   ├── protocol.md
│   ├── dht.md
│   ├── storage.md
│   └── performance.md
│
├── scripts/
│   ├── benchmark.sh
│   └── integration.sh
│
├── Makefile
├── go.mod
├── go.sum
├── README.md
└── LICENSE
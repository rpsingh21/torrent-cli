package piece

type Block struct {
	Piece       int
	Offset      int
	Length      int
	RequestedBy string
	Completed   bool
}

func (b *Block) resetDownload() {
	b.RequestedBy = ""
	b.Completed = false
}

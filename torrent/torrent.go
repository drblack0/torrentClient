package torrent

import "bitTorrentClient/peers"

type PieceWork struct {
	Index  int
	Hash   [20]byte
	Length int
}

type Torrent struct {
	Peers       []peers.Peer
	PeerID      [20]byte
	InfoHash    [20]byte
	PieceHashes [][20]byte // The list of all piece hashes
	PieceLength int
	TotalLength int
	Name        string
	// This is the most important part:
	WorkQueue chan *PieceWork // A channel to hand out work to workers
}

func New(peerId [20]byte, infoHash [20]byte, pieceLenght int, totalLength int, peers []peers.Peer, name string, pieceHashes [][20]byte) *Torrent {
	return &Torrent{
		Peers:       peers,
		PeerID:      peerId,
		InfoHash:    infoHash,
		TotalLength: totalLength,
		PieceLength: pieceLenght,
		Name:        name,
		PieceHashes: pieceHashes,
	}
}

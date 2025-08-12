package client

import (
	"bitTorrentClient/torrent"
	"bytes"
	"crypto/sha1"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"sync"
	"time"
)

type Client struct {
	wg *sync.WaitGroup

	Conn     net.Conn
	Choked   bool
	PeerId   [20]byte
	InfoHash [20]byte
	Address  string
	Bitfield BitField

	// the following fields will be used for downloading pieces
	DataBuffer  []byte
	PieceLength int
	CurrentWork *torrent.PieceWork
	WorkQueue   chan *torrent.PieceWork
	Requested   int
	Downloaded  int
}

func New(infohash [20]byte, address string, peerId [20]byte, dataBuffer []byte, pieceLength int, workQueue chan *torrent.PieceWork, waitgroup *sync.WaitGroup) *Client {
	return &Client{
		wg: waitgroup,

		Choked:   true, // our peer starts in a choked state already
		InfoHash: infohash,
		PeerId:   peerId,
		Address:  address,

		// this is used for dowloading the different pieces
		DataBuffer:  dataBuffer,
		PieceLength: pieceLength,
		WorkQueue:   workQueue,
	}
}

func (c *Client) Run() error {
	defer c.wg.Done()
	handshake := &Handshake{
		Pstr:     "BitTorrent protocol",
		InfoHash: c.InfoHash,
		PeerID:   c.PeerId,
	}

	serializedMsg := handshake.Serialize()

	conn, err := net.DialTimeout("tcp", c.Address, 5*time.Second)
	if err != nil {
		return err
	}

	c.Conn = conn
	fmt.Printf("peer %s: connected\n", c.Address)
	conn.Write(serializedMsg)

	handshakeResp := make([]byte, 68)

	_, err = io.ReadFull(conn, handshakeResp)

	if err != nil {
		return err
	}
	fmt.Printf("peer %s: handshake OK\n", c.Address)

	// Send interested message to let peer know we want pieces
	interestedMsg := &Message{ID: MsgInterested, Payload: []byte{}}
	interestedSerialized := interestedMsg.Serialize()
	conn.Write(interestedSerialized)
	fmt.Printf("peer %s: interested sent\n", c.Address)

	for {
		message, err := Read(conn)

		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}

		if message == nil {
			continue
		}

		switch message.ID {
		case MsgChoke:
			c.Choked = true
			fmt.Printf("peer %s: choked\n", c.Address)
		case MsgUnchoke:
			c.Choked = false
			fmt.Printf("peer %s: unchoked\n", c.Address)

			if c.CurrentWork == nil {
				c.CurrentWork = <-c.WorkQueue
				fmt.Printf("peer %s: assigned piece #%d len=%d\n", c.Address, c.CurrentWork.Index, c.CurrentWork.Length)
				c.requestNextBlock()
			}

		case MsgBitfield:
			c.Bitfield = message.Payload
			fmt.Printf("peer %s: bitfield %d bytes\n", c.Address, len(message.Payload))

		case MsgPiece:
			if len(message.Payload) < 8 {
				continue
			}

			index := binary.BigEndian.Uint32(message.Payload[0:4])
			begin := binary.BigEndian.Uint32(message.Payload[4:8])

			offset := (int(index) * c.PieceLength) + int(begin)
			data := message.Payload[8:]

			copy(c.DataBuffer[offset:], data)
			fmt.Printf("copy_ok peer=%s index=%d begin=%d bytes=%d\n", c.Address, index, begin, len(data))

			c.Downloaded++

			totalBlocks := (c.CurrentWork.Length + 16383) / 16384
			fmt.Printf("piece_progress peer=%s index=%d %d/%d blocks\n", c.Address, c.CurrentWork.Index, c.Downloaded, totalBlocks)

			if c.Downloaded == totalBlocks {
				fmt.Printf("piece_done peer=%s index=%d verifying\n", c.Address, c.CurrentWork.Index)

				// VERIFY THE HASH
				pieceData := c.DataBuffer[c.CurrentWork.Index*c.PieceLength : (c.CurrentWork.Index*c.PieceLength)+c.CurrentWork.Length]
				hash := sha1.Sum(pieceData)

				if bytes.Equal(hash[:], c.CurrentWork.Hash[:]) {
					fmt.Printf("piece_valid peer=%s index=%d\n", c.Address, c.CurrentWork.Index)

					// Get the next job
					c.CurrentWork = <-c.WorkQueue
					c.Requested = 0
					c.Downloaded = 0
					fmt.Printf("peer %s: assigned piece #%d len=%d\n", c.Address, c.CurrentWork.Index, c.CurrentWork.Length)
					c.requestNextBlock()

				} else {
					fmt.Printf("piece_invalid peer=%s index=%d requeue\n", c.Address, c.CurrentWork.Index)
					c.WorkQueue <- c.CurrentWork
					c.CurrentWork = nil // Become idle
					c.Downloaded = 0
					c.Requested = 0
				}
			} else {
				c.requestNextBlock()
			}
		default:
		}
	}
}

func (c *Client) sendRequest(index, begin, length int32) error {
	payload := make([]byte, 12)

	binary.BigEndian.PutUint32(payload[0:4], uint32(index))
	binary.BigEndian.PutUint32(payload[4:8], uint32(begin))
	binary.BigEndian.PutUint32(payload[8:12], uint32(length))

	message := &Message{
		ID:      MsgRequest,
		Payload: payload,
	}

	serializedMsg := message.Serialize()

	_, err := c.Conn.Write(serializedMsg)

	if err != nil {
		fmt.Println("error while sending request to peer: ", err)
		return err
	} else {
		fmt.Printf("Sent request - Index: %d, Begin: %d, Length: %d\n", index, begin, length)
	}
	return nil
}

func (c *Client) requestNextBlock() {

	if c.CurrentWork == nil {
		return
	}

	begin := c.Requested * 16384
	length := 16384

	if begin+length > c.CurrentWork.Length {
		length = c.CurrentWork.Length - begin
	}

	err := c.sendRequest(int32(c.CurrentWork.Index), int32(begin), int32(length))

	if err != nil {
		c.WorkQueue <- c.CurrentWork
		c.CurrentWork = nil
		c.Requested = 0
		c.Downloaded = 0
		return
	}

	c.Requested++
}

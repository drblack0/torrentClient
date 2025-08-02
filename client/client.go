package client

import (
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"time"
)

type Client struct {
	Conn     net.Conn
	Choked   bool
	PeerId   [20]byte
	InfoHash [20]byte
	Address  string
	Bitfield BitField
}

func New(infohash [20]byte, address string, peerId [20]byte) *Client {
	return &Client{
		Choked:   true, // our peer starts in a choked state already
		InfoHash: infohash,
		PeerId:   peerId,
		Address:  address,
	}
}

func (c *Client) Run() error {
	handshake := &Handshake{
		Pstr:     "BitTorrent protocol",
		InfoHash: c.InfoHash,
		PeerID:   c.PeerId,
	}

	serializedMsg := handshake.Serialize()

	conn, err := net.DialTimeout("tcp", c.Address, 5*time.Second)

	if err != nil {
		fmt.Println("error while dialing tcp connection: ", err)
		return err
	}

	c.Conn = conn
	conn.Write(serializedMsg)

	handshakeResp := make([]byte, 68)

	_, err = io.ReadFull(conn, handshakeResp)

	if err != nil {
		fmt.Println("error while reading the handshake resp: ", err)
		return err
	}

	for {
		message, err := Read(conn)

		if err == io.EOF {
			return nil
		}
		if err != nil {
			fmt.Println("error while reading the message: ", err)
			return err
		}

		if message == nil {
			continue
		}

		switch message.ID {
		case MsgChoke:
			c.Choked = true
			fmt.Println("Received: Choke")
		case MsgUnchoke:
			c.Choked = false

			
			if c.Bitfield.HasPiece(0) {
				fmt.Println("attempting to download 0th bit")
				c.sendRequest(0, 0, 16384)
			}
		case MsgInterested:
			fmt.Println("Received: Interested")
		case MsgNotInterested:
			fmt.Println("Received: Not Interested")
		case MsgHave:
			fmt.Println("Received: Have")
		case MsgBitfield:
			c.Bitfield = message.Payload
			fmt.Println("Received: Bitfield")
		case MsgRequest:
			fmt.Println("Received: Request")
		case MsgPiece:
			fmt.Println("Received: Piece")
			if len(message.Payload) < 8 {
				fmt.Println("malformed bytes recieved")
			}

			index := binary.BigEndian.Uint32(message.Payload[0:4])
			beging := binary.BigEndian.Uint32(message.Payload[4:8])

			fmt.Println("here is the index and being: ", index, beging)
		case MsgCancel:
			fmt.Println("Received: Cancel")
		default:
			fmt.Printf("Received: Unknown message ID %d\n", message.ID)
		}
	}
}

func (c *Client) sendRequest(index, begin, length int32) {
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
	}
}

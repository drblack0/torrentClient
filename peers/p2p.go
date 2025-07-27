package peers

import (
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"time"
)

type Peer struct {
	IP   net.IP
	Port uint16
}

func Unmarshal(peersData interface{}) ([]Peer, error) {
	switch peersValue := peersData.(type) {

	// Case 1: Binary Model (most common)
	case string:
		const peerSize = 6
		peersBin := []byte(peersValue)

		if len(peersBin)%peerSize != 0 {
			return nil, fmt.Errorf("received malformed binary peers")
		}
		numPeers := len(peersBin) / peerSize
		peers := make([]Peer, numPeers)

		for i := 0; i < numPeers; i++ {
			offset := i * peerSize
			peers[i].IP = net.IP(peersBin[offset : offset+4])
			peers[i].Port = binary.BigEndian.Uint16(peersBin[offset+4 : offset+6])
		}
		return peers, nil

	// Case 2: Dictionary Model (what you received)
	case []interface{}:
		peers := make([]Peer, 0, len(peersValue))
		for _, item := range peersValue {
			peerMap, ok := item.(map[string]interface{})
			if !ok {
				// Skip malformed entries
				continue
			}

			ipStr, ipOk := peerMap["ip"].(string)
			portVal, portOk := peerMap["port"].(int64)

			if !ipOk || !portOk {
				// Skip malformed entries
				continue
			}

			peer := Peer{
				IP:   net.ParseIP(ipStr),
				Port: uint16(portVal),
			}
			peers = append(peers, peer)
		}
		return peers, nil
	}

	return nil, fmt.Errorf("peers field is in an unexpected format: %T", peersData)
}

func ConnectToPeer(infoHash [20]byte, peerId [20]byte, peerIp string) error {
	handshake := &Handshake{
		Pstr:     "BitTorrent protocol",
		InfoHash: infoHash,
		PeerID:   peerId,
	}

	serializedMsg := handshake.Serialize()

	conn, err := net.DialTimeout("tcp", peerIp, 5*time.Second)

	if err != nil {
		fmt.Println("error while dialing tcp connection: ", err)
		return err
	}

	conn.Write(serializedMsg)

	handshakeResp := make([]byte, 68)

	_, err = io.ReadFull(conn, handshakeResp)

	if err != nil {
		fmt.Println("error while reading the handshake resp: ", err)
		return err
	}

	for {

		fmt.Println("here in the for loop")
		lengthPrefix := make([]byte, 4)

		_, err = io.ReadFull(conn, lengthPrefix)

		if err != nil {
			fmt.Println("error while reading the lenght prefix: ", err)
		}
		lenght := binary.BigEndian.Uint32(lengthPrefix)

		if lenght == 0 {
			fmt.Println("got keep alive from the client, continuing")
			continue
		}
		messageBuf := make([]byte, lenght)

		_, err = io.ReadFull(conn, messageBuf)

		if err != nil {
			fmt.Println("error while reading complete messageId and payload: ", err)
		}

		fmt.Println("testing to see the message id: ", string(messageBuf[0]))
	}
}

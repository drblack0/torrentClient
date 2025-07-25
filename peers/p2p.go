package peers

import (
	"encoding/binary"
	"fmt"
	"net"
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

package main

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"time"

	"bitTorrentClient/bencode"
	"bitTorrentClient/peers"
	"bitTorrentClient/torrentFile"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("correct way to use this is go run . <path-to-file>")
	}

	tf, err := torrentFile.Open(os.Args[1])

	if err != nil {
		fmt.Println("error is here: ", err)
		os.Exit(1)
	}

	// fmt.Println("tf: ", tf.Info)

	hash, err := tf.GetInfoHash()

	fmt.Println("infohash: ", hex.EncodeToString(hash))

	fmt.Println("hash error: ", err)
	trackerUrl, err := tf.BuildTrackerUrl([20]byte(hash))

	if err != nil {
		fmt.Println("error while getting the tracker url: ", trackerUrl)
		os.Exit(1)
	}

	_ = http.Client{Timeout: 15 * time.Second}

	resp, err := http.Get(trackerUrl)

	defer func() {
		err := resp.Body.Close()
		if err != nil {
			fmt.Println("error while closing resp body: ", err)
		}
	}()

	decodedResponse, err := bencode.Decode(resp.Body)

	peerList := decodedResponse.(map[string]interface{})["peers"]

	peerListDecoded, err := peers.Unmarshal(peerList)

	for _, item := range peerListDecoded {
		peerAddress := fmt.Sprintf("%s:%d", item.IP, item.Port)

		peers.ConnectToPeer([20]byte(hash), [20]byte(tf.PeerId), peerAddress)

		handshake := &peers.Handshake{
			Pstr:     "BitTorrent protocol",
			InfoHash: [20]byte(hash),
			PeerID:   [20]byte(tf.PeerId),
		}

		serializedMsg := handshake.Serialize()

		conn, err := net.DialTimeout("tcp", peerAddress, 5*time.Second)

		if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
			fmt.Printf("Connection to %s timed out. Trying next.\n", peerAddress)
			continue // This was a timeout, it's expected, so we just move on.
		}

		conn.Write(serializedMsg)

		handshakeResp := make([]byte, 68)

		_, err = io.ReadFull(conn, handshakeResp)

		if err != nil {
			fmt.Println("error while reading the handshake resp: ", err)
			return
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
}

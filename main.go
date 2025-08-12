package main

import (
	"encoding/hex"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"bitTorrentClient/bencode"
	"bitTorrentClient/client"
	"bitTorrentClient/peers"
	"bitTorrentClient/torrentFile"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("correct way to use this is go run . <path-to-file>")
	}

	tf, err := torrentFile.Open(os.Args[1])

	if err != nil {
		fmt.Println("open_err:", err)
		os.Exit(1)
	}

	// fmt.Println("tf: ", tf.Info)

	hash, err := tf.GetInfoHash()
	if err != nil {
		fmt.Println("infohash_err:", err)
		os.Exit(1)
	}
	fmt.Printf("torrent name=%s pieces=%d piece_len=%d size=%d\n", tf.Info.Name, len(tf.GetPieceHashes()), tf.Info.PieceLength, tf.CalculateSize())
	fmt.Println("infohash:", hex.EncodeToString(hash))
	trackerUrl, err := tf.BuildTrackerUrl([20]byte(hash))

	if err != nil {
		fmt.Println("tracker_url_err:", err)
		os.Exit(1)
	}

	_ = http.Client{Timeout: 15 * time.Second}

	resp, err := http.Get(trackerUrl)

	defer func() {
		err := resp.Body.Close()
		if err != nil {
			fmt.Println("resp_close_err:", err)
		}
	}()

	decodedResponse, err := bencode.Decode(resp.Body)
	if err != nil {
		fmt.Println("bencode_err:", err)
		os.Exit(1)
	}

	peerList := decodedResponse.(map[string]interface{})["peers"]
	peerListDecoded, err := peers.Unmarshal(peerList)
	if err != nil {
		fmt.Println("peers_unmarshal_err:", err)
		os.Exit(1)
	}
	fmt.Printf("tracker: %d peers\n", len(peerListDecoded))

	pieceHashes := tf.GetPieceHashes()
	dataBuffer := make([]byte, tf.CalculateSize())
	workQueue := tf.CreateWorkQueue(pieceHashes, tf.CalculateSize())

	var wg sync.WaitGroup
	// fmt.Println("here are the piece Hashes: ", hashses)
	fmt.Printf("starting %d peers\n", len(peerListDecoded))
	for _, item := range peerListDecoded {
		wg.Add(1)
		addr := fmt.Sprintf("%s:%d", item.IP.String(), item.Port)
		fmt.Printf("spawn peer %s\n", addr)
		client := client.New([20]byte(hash), addr, [20]byte(tf.PeerId), dataBuffer, int(tf.Info.PieceLength), workQueue, &wg)
		go client.Run()
	}

	wg.Wait()

	fmt.Println("download_complete: saving to disk...")
	err = os.WriteFile(tf.Info.Name, dataBuffer, 0644)
	if err != nil {
		log.Fatalf("Failed to save file: %v", err)
	}
	fmt.Printf("saved file=%s\n", tf.Info.Name)

}

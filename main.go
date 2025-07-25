package main

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"bitTorrentClient/bencode"
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

	jsonBytes, err := json.MarshalIndent(decodedResponse, "", " ")

	if err != nil {
		fmt.Println("error while marshalling: ", err)
	}

	fmt.Println(string(jsonBytes))

}

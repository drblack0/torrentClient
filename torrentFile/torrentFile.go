package torrentFile

import (
	"bytes"
	"crypto/rand"
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"

	"bitTorrentClient/bencode"
)

// In torrentfile.go
type Info struct {
	PieceLength int64      `bencode:"piece length" json:"piece length"`
	Pieces      string     `bencode:"pieces"       json:"pieces"`
	Name        string     `bencode:"name"         json:"name"`
	Length      int64      `bencode:"length"       json:"length,omitempty"` // omitempty is good practice
	Files       []FileInfo `bencode:"files"        json:"files,omitempty"`
}

type FileInfo struct {
	Length int64    `bencode:"length" json:"length"`
	Path   []string `bencode:"path"   json:"path"`
}

type TorrentFile struct {
	Announce     string     `bencode:"announce"       json:"announce"`
	Info         Info       `bencode:"info"           json:"info"`
	AnnounceList [][]string `bencode:"announce-list"  json:"announce-list"`
	CreationDate int64      `bencode:"creation date"  json:"creation date"`
	Comment      string     `bencode:"comment"        json:"comment"`
	CreatedBy    string     `bencode:"created by"     json:"created by"`
	Encoding     string     `bencode:"encoding"       json:"encoding"`
}

func Open(path string) (*TorrentFile, error) {
	fileReader, err := os.Open(path)

	if err != nil {
		return nil, fmt.Errorf("error while reading a file %v", err)
	}

	decodedData, err := bencode.Decode(fileReader)

	if err != nil {
		return nil, fmt.Errorf("error while decoding the data: %v", err)
	}

	jsonBytes, err := json.Marshal(decodedData)
	res := &TorrentFile{}

	if err != nil {
		return nil, fmt.Errorf("error while marshalling the decoded data: %v", err)
	}

	err = json.Unmarshal(jsonBytes, res)

	if err != nil {
		return nil, fmt.Errorf("error while unmarshalling the data %v", err)
	}

	return res, nil
}

func (tf *TorrentFile) GetInfoHash(info map[string]interface{}) ([]byte, error) {
	// var res map[string]interface{}

	// jsonBytes, err := json.Marshal(info)

	// if err != nil {
	// 	return nil, err
	// }

	// err = json.Unmarshal(jsonBytes, &res)

	// if err != nil {
	// 	return nil, err
	// }

	var buf bytes.Buffer

	err := bencode.Encode(&buf, info)

	if err != nil {
		return nil, err
	}

	shaSum := sha1.Sum(buf.Bytes())
	return shaSum[:], nil
}

func (tf *TorrentFile) BuildTrackerUrl(infoHash [20]byte) (string, error) {
	params := url.Values{}
	params.Add("info_hash", string(infoHash[:]))

	peerId := make([]byte, 20)

	_, err := rand.Read(peerId)

	if err != nil {
		return "", err
	}

	var left int

	for _, item := range tf.Info.Files {
		left += int(item.Length)
	}

	if left == 0 {
		left += int(tf.Info.Length)
	}

	params.Add("peer_id", string(peerId[:]))
	params.Add("port", "6881")
	params.Add("uploaded", "0")
	params.Add("downloaded", "0")
	params.Add("left", strconv.Itoa(left))

	var announce string

	if strings.HasPrefix(tf.Announce, "udp") {
		announce = tf.getAnnounceUrl()
	} else {
		announce = tf.Announce
	}

	trackerUrl := fmt.Sprintf("%s?%s", announce, params.Encode())

	return trackerUrl, nil
}

func (tf *TorrentFile) getAnnounceUrl() string {

	for _, tier := range tf.AnnounceList {
		for _, item := range tier {
			if strings.HasPrefix(item, "http") {
				fmt.Println("announce: ", item)
				return item
			}
		}
	}

	fmt.Println("there is no http announce url")
	return ""
}

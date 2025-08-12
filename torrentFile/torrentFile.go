package torrentFile

import (
	"bytes"
	"crypto/rand"
	"crypto/sha1"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"

	"bitTorrentClient/bencode"
	"bitTorrentClient/torrent"
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
	InfoBytes    []byte     `bencode:"infoBytes"	  json:"infoBytes"`
	PeerId       []byte     `bencode:"peerId" json:"peerId`
}

func Open(path string) (*TorrentFile, error) {
	fileReader, err := os.Open(path)

	if err != nil {
		return nil, fmt.Errorf("error while reading a file %v", err)
	}

	decodedData, err := bencode.Decode(fileReader)

	infoMap, ok := decodedData.(map[string]interface{})["info"]

	if !ok {
		return nil, fmt.Errorf("error while asserting info map")
	}

	var infoBuf bytes.Buffer
	err = bencode.Encode(&infoBuf, infoMap)

	if err != nil {
		return nil, fmt.Errorf("error while decoding the data: %v", err)
	}

	res, err := toTorrentFile(decodedData.(map[string]interface{}))

	if err != nil {
		return nil, fmt.Errorf("error while populating torrentfile %v", err)
	}

	res.InfoBytes = infoBuf.Bytes()

	return res, nil
}

// toTorrentFile manually maps a decoded bencode map to a TorrentFile struct.
// This is a robust way to create the struct without data corruption from other formats like JSON.
func toTorrentFile(data map[string]interface{}) (*TorrentFile, error) {
	var tf TorrentFile

	// --- 1. Populate top-level string and integer fields ---
	// We use safe type assertions. If a key doesn't exist or has the wrong type,
	// the field will be left as its zero value (e.g., empty string or 0).
	if announce, ok := data["announce"].(string); ok {
		tf.Announce = announce
	}
	if comment, ok := data["comment"].(string); ok {
		tf.Comment = comment
	}
	if createdBy, ok := data["created by"].(string); ok {
		tf.CreatedBy = createdBy
	}
	if creationDate, ok := data["creation date"].(int64); ok {
		tf.CreationDate = creationDate
	}
	if encoding, ok := data["encoding"].(string); ok {
		tf.Encoding = encoding
	}

	// --- 2. Populate the nested 'info' dictionary ---
	infoMap, ok := data["info"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("missing or invalid 'info' dictionary in torrent file")
	}

	if name, ok := infoMap["name"].(string); ok {
		tf.Info.Name = name
	}
	if pl, ok := infoMap["piece length"].(int64); ok {
		tf.Info.PieceLength = pl
	}
	// This is the most critical part: get the raw, uncorrupted binary string
	if pieces, ok := infoMap["pieces"].(string); ok {
		tf.Info.Pieces = pieces
	}

	// Handle both single-file and multi-file torrents
	if length, ok := infoMap["length"].(int64); ok {
		tf.Info.Length = length // Single-file torrent
	}

	if filesData, ok := infoMap["files"].([]interface{}); ok {
		// Multi-file torrent, loop through the files list
		for _, fileData := range filesData {
			fileMap, ok := fileData.(map[string]interface{})
			if !ok {
				continue // Skip malformed entries
			}

			var fileInfo FileInfo
			if l, ok := fileMap["length"].(int64); ok {
				fileInfo.Length = l
			}

			// The 'path' is a list of interfaces, where each should be a string
			if pathData, ok := fileMap["path"].([]interface{}); ok {
				for _, pathElement := range pathData {
					if pathStr, ok := pathElement.(string); ok {
						fileInfo.Path = append(fileInfo.Path, pathStr)
					}
				}
			}
			tf.Info.Files = append(tf.Info.Files, fileInfo)
		}
	}

	// --- 3. Populate the 'announce-list' (list of lists of strings) ---
	if announceList, ok := data["announce-list"].([]interface{}); ok {
		for _, tierData := range announceList {
			tier, ok := tierData.([]interface{})
			if !ok {
				continue // Skip malformed tiers
			}

			var tierList []string
			for _, trackerData := range tier {
				if tracker, ok := trackerData.(string); ok {
					tierList = append(tierList, tracker)
				}
			}
			tf.AnnounceList = append(tf.AnnounceList, tierList)
		}
	}

	return &tf, nil
}

func (tf *TorrentFile) GetInfoHash() ([]byte, error) {

	shaSum := sha1.Sum(tf.InfoBytes)
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

	tf.PeerId = peerId

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

func (tf *TorrentFile) CalculateSize() int {
	var totalLength int

	for _, file := range tf.Info.Files {
		totalLength += int(file.Length)
	}

	if totalLength == 0 {
		totalLength = int(tf.Info.Length)
	}

	return totalLength
}

func (tf *TorrentFile) GetPieceHashes() [][20]byte {
	var res [][20]byte

	if len(tf.Info.Pieces)%20 != 0 {
		fmt.Println("corrupted file")
		return nil
	}

	numTimes := len(tf.Info.Pieces) / 20

	for i := 0; i < numTimes; i++ {
		var hash [20]byte

		hashSlice := []byte(tf.Info.Pieces[i*20 : (i+1)*20]) // (i+1)*20 is a bit safer

		copy(hash[:], hashSlice)

		res = append(res, hash)
	}

	return res
}

func (tf *TorrentFile) CreateWorkQueue(pieceHashes [][20]byte, totalSize int) chan *torrent.PieceWork {
	pieceWorks := make([]*torrent.PieceWork, len(pieceHashes))
	for i, hash := range pieceHashes {
		// Calculate the length of this specific piece
		begin := i * int(tf.Info.PieceLength)
		end := begin + int(tf.Info.PieceLength)
		if end > totalSize {
			end = totalSize
		}
		length := end - begin

		pieceWorks[i] = &torrent.PieceWork{
			Index:  i,
			Hash:   hash,
			Length: length,
		}
	}

	workQueue := make(chan *torrent.PieceWork, len(pieceHashes))

	for _, work := range pieceWorks {
		workQueue <- work
	}

	return workQueue
}

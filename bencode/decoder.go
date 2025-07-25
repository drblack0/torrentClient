package bencode

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
)

func Decode(r io.Reader) (interface{}, error) {
	reader := bufio.NewReader(r)
	return decodeRecursive(reader)
}

func decodeRecursive(r *bufio.Reader) (interface{}, error) {
	firstByte, err := r.Peek(1)
	if err != nil {
		return nil, fmt.Errorf("error while peeking into the buffer")
	}

	switch {
	case firstByte[0] == 'i':
		return decodeInt(r)
	case firstByte[0] >= '0' && firstByte[0] <= '9':
		return decodeString(r)
	case firstByte[0] == 'l':
		return decodeList(r)
	case firstByte[0] == 'd':
		return decodeDict(r)
	default:
		return nil, fmt.Errorf("invalid bencode type: got %c", firstByte[0])
	}
}

func decodeInt(r *bufio.Reader) (int64, error) {

	_, err := r.ReadByte()
	if err != nil {
		return 0, fmt.Errorf("error while decoding int")
	}

	byteStrings, err := r.ReadString('e')

	result, err := strconv.ParseInt(byteStrings[:len(byteStrings)-1], 10, 64)

	if err != nil {
		return 0, fmt.Errorf("error while decoding int")
	}

	return result, nil
}

func decodeString(r *bufio.Reader) (string, error) {
	lengthString, err := r.ReadString(':')
	if err != nil {
		return "", fmt.Errorf("error while reading string: ")
	}

	length, err := strconv.ParseInt(lengthString[:len(lengthString)-1], 10, 64)

	if err != nil {
		return "", fmt.Errorf("error while parsing integer")
	}
	bytes := make([]byte, length)

	_, err = io.ReadFull(r, bytes)

	if err != nil {
		return "", fmt.Errorf("error while reading into the bytes")
	}

	return string(bytes), nil
}

func decodeList(r *bufio.Reader) ([]interface{}, error) {
	var res []interface{}

	r.ReadByte()
	for {
		peekedByte, err := r.Peek(1)

		if err != nil {
			return nil, fmt.Errorf("error while peeking")
		}
		if peekedByte[0] == 'e' {
			break
		}

		currentEle, err := decodeRecursive(r)

		if err != nil {
			return nil, fmt.Errorf("error while getting recursive result")
		}

		res = append(res, currentEle)
	}

	r.ReadByte()
	return res, nil
}

func decodeDict(r *bufio.Reader) (map[string]interface{}, error) {
	res := make(map[string]interface{}, 0)

	r.ReadByte()

	for {
		peekedByte, err := r.Peek(1)

		if err != nil {
			return nil, fmt.Errorf("error while peeking the bytes")
		}

		if peekedByte[0] == 'e' {
			break
		}

		key, err := decodeString(r)

		if err != nil {
			return nil, fmt.Errorf("error while decoding key for dictionary")
		}

		value, err := decodeRecursive(r)

		if err != nil {
			return nil, fmt.Errorf("error while parsing value")
		}

		res[key] = value
	}

	r.ReadByte()
	return res, nil
}

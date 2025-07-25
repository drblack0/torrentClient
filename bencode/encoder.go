package bencode

import (
	"fmt"
	"io"
	"slices"
)

func Encode(w io.Writer, v interface{}) error {

	switch val := v.(type) {
	case string:
		return encodeString(w, val)
	case int64:
		return encodeInt(w, val)
	case []interface{}:
		return encodeList(w, val)
	case map[string]interface{}:
		return encodeDict(w, val)
	}
	return nil
}

func encodeString(w io.Writer, val string) error {
	_, err := w.Write([]byte(fmt.Sprintf("%d:%s", len(val), val)))

	if err != nil {
		fmt.Println("error while encoding the string: ", err)
		return err
	}
	return nil
}

func encodeInt(w io.Writer, v int64) error {
	_, err := w.Write([]byte(fmt.Sprintf("i%de", v)))

	if err != nil {
		return fmt.Errorf("error while parsing integer: %v", err)
	}

	return nil
}

func encodeList(w io.Writer, v []interface{}) error {
	_, err := w.Write([]byte("l"))
	if err != nil {
		return err
	}

	for _, item := range v {
		err := Encode(w, item)

		if err != nil {
			return err
		}
	}

	_, err = w.Write([]byte("e"))

	if err != nil {
		return err
	}

	return nil
}

func encodeDict(w io.Writer, v map[string]interface{}) error {
	_, err := w.Write([]byte("d"))

	if err != nil {
		return err
	}

	var tempSlice []string
	for key, _ := range v {
		tempSlice = append(tempSlice, key)
	}

	slices.Sort(tempSlice)

	for _, item := range tempSlice {
		err := encodeString(w, item)
		if err != nil {
			return err
		}

		err = Encode(w, v[item])

		if err != nil {
			return err
		}
	}

	_, err = w.Write([]byte("e"))
	if err != nil {
		return err
	}
	return nil
}

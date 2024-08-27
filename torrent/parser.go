package bencode

import (
	"bufio"
	"io"
)

func Parse(r io.Reader) (*BObject, error) {
	br, ok := r.(*bufio.Reader)
	if !ok {
		br = bufio.NewReader(r)
	}
	// recursively decreasing
	b, err := br.Peek(1) // read the peek without advancing the reader's position
	if err != nil {
		return nil, err
	}
	var res BObject
	switch {
	case b[0] >= '0' && b[0] <= '9':
		// string
		val, err := DecodeString(br)
		if err != nil {
			return nil, err
		}
		res.type_ = BSTR
		res.val_ = val
	case b[0] == 'i':
		// int
		val, err := DecodeInt(br)
		if err != nil {
			return nil, err
		}
		res.type_ = BINT
		res.val_ = val
	case b[0] == 'l':
		// list
		br.ReadByte() // read and consume a single byte `l`, advancing the position
		var list []*BObject
		for {
			if p, _ := br.Peek(1); p[0] == 'e' { // p is a 1-length slice of byte `[]byte`
				br.ReadByte()
				break
			}
			elem, err := Parse(br) // recursive parsing
			if err != nil {
				return nil, err
			}
			list = append(list, elem)
		}
		res.type_ = BLIST
		res.val_ = list
	case b[0] == 'd':
		// map
		br.ReadByte()
		dict := make(map[string]*BObject)
		for {
			if p, _ := br.Peek(1); p[0] == 'e' {
				br.ReadByte()
				break
			}
			key, err := DecodeString(br)
			if err !=  nil {
				return nil, err
			}
			val, err := Parse(br)
			if err != nil {
				return nil, err
			}
			dict[key] = val
		}
		res.type_ = BDICT
		res.val_ = dict
	default:
		return nil, ErrIvd
	}
	return &res, nil
}
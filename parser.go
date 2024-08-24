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
	b, err := br.Peek(1)
	if err != nil {
		return nil, err
	}
	var ret BObject
	switch {
	case b[0] >= '0' && b[0] <= '9':
		// string
		val, err := DecodeString(br)
		if err != nil {
			return nil, err
		}
		ret.type_ = BSTR
		ret.val_ = val
	case b[0] == 'i':
		// int
		val, err := DecodeInt(br)
		if err != nil {
			return nil, err
		}
		ret.type_ = BINT
		ret.val_ = val
	
	}
}

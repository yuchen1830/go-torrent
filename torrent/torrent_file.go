package torrent

import (
	"bytes"
	"crypto/shal"
	"fmt"
	"go-torrent/bencode"
	"io"
)

// torrent file: announce + info(name, length, pieces, piece length)
// 3 structs with tags: 1.rawFile(2) 2.info(5) 3.torrentFile(6)
type rawFile struct{
	Announce	string `bencode:"announce"`
	Info	 	rawInfo `bencode:"info"`
}

type rawInfo struct {
	Name		string `bencode:"name"`	
	Length		int	`bencode:"length"`
	Pieces		string `bencode:"pieces"`
	PieceLength	int `bencode:"piece length"`
}

const SHALEN int = 20

type TorrentFile struct {
	Announce	string
	InfoSHA		[SHALEN]byte // <- tracker
	FileName	string
	FileLen		int
	PieceLen	int
	PieceSHA	[][SHALEN]byte
}

func ParseFile(r io.Reader) (*TorrentFile, error) {
	raw := new(rawFile)
	err := bencode.Unmarshal(r, raw)
	if err != nil {
		fmt.Println("Fail to parse torrent file")
		return nil, err
	}
	// raw file -> torrent file
	res := new(TorrentFile)
	res.Announce = raw.Announce
	res.FileName = raw.Info.Name
	res.FileLen = raw.Info.Length
	res.PieceLen = raw.Info.PieceLength

	// SHA-1
	// TAG NOT USED?
	buf := new(bytes.Buffer)
	wlen := bencode.Marshal(buf, raw.Info)
	// encodes raw.Info into the Bencode format and writes it into the buffer
	if wlen == 0 {
		fmt.Println("raw file into error")
	}
	res.InfoSHA = shal.Sum(buf.Bytes())
	// The buf.Bytes() method returns the byte slice of the buffer
	// computes the SHA-1 hash of these bytes

	bys := []byte(raw.Info.Pieces)
	cnt := len(bys) / SHALEN
	// calculates how many SHA-1 hashes are contained within bys
	hashes := make([][SHALEN]byte, cnt)
	// stores individual hashes of the pieces, has cnt elments, each element is 20-byte array
	for i := 0; i < cnt; i++ {
		copy(hashes[i][:], bys[i*SHALEN:(i+1)*SHALEN])
	}
	res.PieceSHA = hashes
	return res, nil
}

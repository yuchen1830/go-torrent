package torrent

import (
	"fmt"
	"io"
)

type HandshakeMsg struct {
	PreStr  string
	InfoSHA [SHALEN]byte
	PeerId  [IDLEN]byte
}

const (
	Reserved int = 8
	HsMsgLen int = Reserved + SHALEN + IDLEN
)

func NewHandShakeMsg(infoSHA [SHALEN]byte, peerId [IDLEN]byte) *HandshakeMsg {
	return &HandshakeMsg{
		PreStr:  "BitTorrent protocol",
		InfoSHA: infoSHA,
		PeerId:  peerId,
	}
}

// components: 1(length of protocol) + protocol + reserved + info
func WriteHandShake(w io.Writer, msg *HandshakeMsg) (int, error) {
	buf := make([]byte, len(msg.PreStr)+HsMsgLen+1)
	buf[0] = byte(len(msg.PreStr))
	curr := 1
	curr += copy(buf[curr:], []byte(msg.PreStr))
	curr += copy(buf[curr:], make([]byte, Reserved))
	curr += copy(buf[curr:], msg.InfoSHA[:])
	curr += copy(buf[curr:], msg.PeerId[:])
	return w.Write(buf)
}

func ReadHandshake(r io.Reader) (*HandshakeMsg, error) {
	lenBuf := make([]byte, 1)
	_, err := io.ReadFull(r, lenBuf)
	if err != nil {
		return nil, err
	}
	
	prelen := int(lenBuf[0]) // protocol length
	if prelen == 0 {
		err := fmt.Errorf("prelen cannot be 0")
		return nil, err
	}
	msgBuf := make([]byte, prelen + HsMsgLen)
	_, err = io.ReadFull(r, msgBuf)
	if err != nil {
		return nil, err
	}

	var InfoSHA [SHALEN]byte
	var PeerId	[IDLEN]byte

	copy(InfoSHA[:], msgBuf[prelen+Reserved : prelen+Reserved+SHALEN])
	copy(PeerId[:], msgBuf[prelen+Reserved+SHALEN : prelen+Reserved+SHALEN+IDLEN])


	return &HandshakeMsg {
		PreStr: 	string(msgBuf[0:prelen]),
		InfoSHA: 	InfoSHA,
		PeerId: 	PeerId,
	}, nil
}
package torrent

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"internal/msan"
	"io"
	"net"
	"strconv"
	"time"

	"golang.org/x/tools/go/expect"
)

type MsgId uint8

const (
	MsgChoke			MsgId = 0
	MsgUnchoke			MsgId = 1
	MsgInterested		MsgId = 2
	MsgNotInterested	MsgId = 3
	MsgHave				MsgId = 4
	MsgBitfield			MsgId = 5
	MsgRequest			MsgId = 6
	MsgPiece			MsgId = 7
	MsgCancel			MsgId = 8
)

type PeerMsg struct {
	Id		MsgId
	Payload []byte
}

type PeerConn struct {
	net.Conn
	Choked		bool
	Field		Bitfield
	Peer		PeerInfo
	peerId		[IDLEN]byte
	InfoSHA		[SHALEN]byte
}

// 1. handshake: 
func handshake(conn net.Conn, infoSHA [SHALEN]byte, peerId [IDLEN]byte) error {
	conn.SetDeadline(time.Now().Add(3 * time.Second))
	defer conn.SetDeadline(time.Time{})
	
	req := NewHandShakeMsg(infoSHA, peerId)
	_, err := WriteHandShake(conn, req)
	if err != nil {
		fmt.Println("send handshake failed")
		return err
	}

	res, err := ReadHandshake(conn)
	if err != nil {
		fmt.Println("read handshake failed")
		return err
	}

	if !bytes.Equal(res.InfoSHA[:], infoSHA[:]){
		fmt.Println("check handshake failed")
		return fmt.Errorf("handshake msg error: " + string(res.InfoSHA[:]))
	}
	return nil
}

// 2. pieces info (bit map)
func fillBitfield(c *PeerConn) error {
	c.SetDeadline(time.Now().Add(5 * time.Second))
	defer c.SetDeadline(time.Time{})

	msg, err := c.ReadMsg()
	if err != nil {
		return err
	}
	if msg == nil {
		return fmt.Errorf("expected bitfield")
	}
	if msg.Id != MsgBitfield {
		return fmt.Errorf("expected bitfield, get " + strconv.Itoa(int(msg.Id)))

	}
	fmt.Println("fill bitfield : " + c.Peer.Ip.String())
	c.Field = msg.Payload
	return nil
}

func (c *PeerConn) ReadMsg() (*PeerMsg, error) {
	// read msg length
	lenBuf := make([]byte, 4)
	_, err := io.ReadFull(c, lenBuf)
	if err != nil {
		return nil, err
	}
	length := binary.BigEndian.Uint32(lenBuf)

	if length == 0 {
		return nil, nil
	}

	msgBuf := make([]byte, length)
	_, err = io.ReadFull(c, msgBuf)
	if err != nil {
		return nil, err
	}
	return &PeerMsg{
		Id:			MsgId(msgBuf[0]),
		Payload: 	msgBuf[1:],
	}, nil
	// PeerMsg: Id(kind) + contents
}

const LenBytes uint32 = 4

func(c *PeerConn) WriteMsg(m *PeerMsg) (int, error) {
	var buf []byte
	if m == nil {
		buf = make([]byte, LenBytes)
	}
	length := uint32(len(m.Payload) + 1) // Id: 1
	buf = make([]byte, LenBytes+length)
	binary.BigEndian.PutUint32(buf[0:LenBytes], length)
	buf[LenBytes] = byte(m.Id)
	copy(buf[LenBytes+1:], m.Payload)
	return c.Write(buf)
}

func CopyPieceData(index int, buf []byte, msg *PeerMsg) (int, error) {
	if msg.Id != MsgPiece {
		
	}
}

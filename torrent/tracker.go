package torrent

import (
	"encoding/binary"
	"fmt"
	"go-torrent/bencode"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

const (
	PeerPort	int = 3456
	IpLen		int = 4
	PortLen		int = 2
	PeerLen		int = IpLen + PortLen 
)

const IDLEN int = 20

type PeerInfo struct {
	Ip		net.IP
	Port	uint16
}

type TrackerResp struct {
	Interval	int		`bencode:"interval"`
	Peers		string	`bencode:"peers"`
}

func buildUrl(tf *TorrentFile, peerId [IDLEN]byte) (string, error) {
	base, err := url.Parse(tf.Announce)
	if err != nil {
		fmt.Println("Announce error: " + tf.Announce)
		return "", err
	}

	params := url.Values {
		"info_hash":	[]string{string(tf.InfoSHA[:])},
		"peer_id":	[]string{string(peerId[:])},
		"port":		[]string{strconv.Itoa(PeerPort)},
		// "uploaded"
		// "downloaded"
		// "compact"
		"left":		[]string{strconv.Itoa(tf.FileLen)},
	}

	base.RawQuery = params.Encode()
	return base.String(), nil
}

func buildPeerInfo(peers []byte) []PeerInfo {
	num := len(peers) / PeerLen
	if len(peers)%PeerLen != 0 {
		fmt.Println("Received malformed peers")
		return nil
	}
	infos := make([]PeerInfo, num)
	for i := 0; i < num; i++ {
		offset := i * PeerLen
		infos[i].Ip = net.IP(peers[offset : offset+IpLen])
		infos[i].Port = binary.BigEndian.Uint16(peers[offset+IpLen : offset+PeerLen])
	}
	return infos
}

func FindPeers(tf *TorrentFile, peerId [IDLEN]byte) []PeerInfo {
	// request
	url, err := buildUrl(tf, peerId)
	if err != nil {
		fmt.Println("Build tracker url error: " + err.Error())
		return nil
	}

	// http GET
	cli := &http.Client{Timeout: 15 * time.Second}
	resp, err := cli.Get(url)
	if err != nil {
		fmt.Println("Fail to connect to track: " + err.Error())
		return nil
	}
	defer resp.Body.Close()

	trackResp := new(TrackerResp)
	err = bencode.Unmarshal(resp.Body, trackResp)
	if err != nil {
		fmt.Println("Tracker response error: " + err.Error())
		return nil
	}

	return buildPeerInfo([]byte(trackResp.Peers))
}
package torrent

import (
	"bytes"
	"crypto/sha1"
	"fmt"
	"os"
	"time"
)


type TorrentTask struct {
	PeerId		[20]byte
	PeerList	[]PeerInfo
	InfoSHA		[SHALEN]byte
	FileName	string
	FileLen		int
	PieceLen	int
	PieceSHA	[][SHALEN]byte // hashes of all pieces, used to verify the integrity of pieces after being downloaded
}

type pieceTask struct {
	index	int
	sha		[SHALEN]byte
	length 	int
}

type taskState struct {
	index		int
	conn  		*PeerConn
	requested	int
	downloaded	int
	backlog		int
	data		[]byte
}

type pieceResult struct {
	index	int
	data	[]byte
}

const	BLOCKSIZE = 15000
const	MAXBLOCK = 5

func Download(task *TorrentTask) error {
	fmt.Println("start downloading " + task.FileName)
	// initialize 2 channels
	taskQueue := make(chan *pieceTask, len(task.PieceSHA))
	resultQueue := make(chan *pieceResult)
	// split torrentTask to pieceTask
	for index, sha := range task.PieceSHA {
		begin, end := task.getPieceBound(index)
		taskQueue <- &pieceTask{index, sha, (end-begin)}
	}
	// initialize goroutine for each peer
	for _, peer := range task.PeerList {
		go task.peerRoutine(peer, taskQueue, resultQueue)
	}
	// each peer, store the data in RAM
	buf := make([]byte, task.FileLen)
	count := 0
	for count < len(task.PieceSHA) {
		res := <- resultQueue
		begin, end := task.getPieceBound(res.index)
		// check in go routine??
		copy(buf[begin:end], res.data)
		count++
		// progress
		percent := float64(count) / float64(len(task.PieceSHA))
		fmt.Printf("downloading, progress: (%0.2%%)\n", percent)
	}
	close(taskQueue)
	close(resultQueue)

	// create file
	file, err := os.Create(task.FileName)
	if err != nil {
		fmt.Println("fail to create file: " + task.FileName)
		return err
	}
	_, err = file.Write(buf)
	if err != nil {
		fmt.Println("fail to write data")
		return err
	}
	return nil
}

func (t *TorrentTask) peerRoutine(peer PeerInfo, taskQueue chan *pieceTask, resultQueue chan *pieceResult) {
	// connect with peer
	conn, err := NewConn(peer, t.InfoSHA, t.PeerId)
	if err != nil {
		fmt.Println("fail to connect peer: " + peer.Ip.String())
		return
	}
	defer conn.Close()

	fmt.Println("successful handshake with peer: " + peer.Ip.String())
	conn.WriteMsg(&PeerMsg{MsgInterested, nil})
	// send the task back to the the task channel with any failure 
	for task := range taskQueue {
		// piece not found
		if !conn.Field.HasPiece(task.index) {
			taskQueue <- task
			continue
		}
		fmt.Printf("get task, index: %v, peer : %v\n", task.index, peer.Ip.String())
		res, err := downloadPiece(conn, task)
		// download failure
		if err != nil {
			taskQueue <- task
			fmt.Println("fail to download piece" + err.Error())
			return
		}
		// check integrity failed
		if !checkPiece(task, res) {
			taskQueue <- task
			continue
		}
		resultQueue <- res
	}
}


func (t *TorrentTask) getPieceBound(index int) (begin, end int) {
	begin = index * t.PieceLen
	end = begin + t.PieceLen
	if end > t.FileLen {
		end = t.FileLen
	}
	return
}

// ??????
func checkPiece(task *pieceTask, res *pieceResult) bool {
	sha := sha1.Sum(res.data)
	if !bytes.Equal(task.sha[:], sha[:]) {
		fmt.Printf("check integrity failed, index: %v\n", res.index)
		return false
	}
	return true
}

func (state *taskState) handleMsg() error {
	msg, err := state.conn.ReadMsg()
	if err != nil {
		return err
	}
	// heartbeat
	if msg == nil {
		return nil
	}
	// otherwise there must be an Id
	switch msg.Id {
	case MsgChoke:
		state.conn.Choked = true // default
	case MsgUnchoke:
		state.conn.Choked = false
	case MsgHave: 
	// onece a new piece is downloaded, the peer sends a Msg and updates bitfield 
		index, err := GetHaveIndex(msg)
		if err != nil {
			return err
		}
		state.conn.Field.SetPiece(index)
	case MsgPiece:
		// request
		n, err := CopyPieceData(state.index, state.data, msg)
		if err != nil {
			return err
		}
		// data update after successful download
		state.downloaded += n
		state.backlog--
	}
	return nil
}

func downloadPiece(conn *PeerConn, task *pieceTask) (*pieceResult, error) {
	state := &taskState{
		index:	task.index,
		conn:	conn,
		data:	make([]byte, task.length),
	}
	conn.SetDeadline(time.Now().Add(15 * time.Second))
	defer conn.SetDeadline(time.Time{})

	for state.downloaded < task.length {
		if !conn.Choked {
			for state.backlog < MAXBLOCK && state.requested < task.length {
				length := BLOCKSIZE
				if task.length - state.requested < length {
					length = task.length - state.requested
				}
				msg := NewRequestMsg(state.index, state.requested, length)
				_, err := state.conn.WriteMsg(msg)
				if err != nil {
					return nil, err
				}
				state.backlog++
				state.requested += length
			}
		}
		err := state.handleMsg()
		if err != nil {
			return nil, err
		}
	}
	return &pieceResult{state.index, state.data}, nil
}


# go-torrent

`go-torrent` is a BitTorrent client implemented in Go. This project includes the capability to parse `.torrent` files, interact with trackers to obtain a list of peers, and download pieces of files concurrently from multiple peers.

## Features

1. **Torrent file parsing:** Utilizes the `bencode` library to parse `.torrent` files, extracting necessary metadata such as the file name, pieces hashes(bit field), and tracker URLs.
2. **Tracker communication**: Communicates with the tracker using the HTTP protocol to obtain a list of available peers.
3. **Peer-to-Peer downloading:** Established TCP connections with peers to download file pieces concurrently, veriyfing their integrity upon reciept.

## Workfolw

### 1. Torrent file parsing

The client begins by parsing a `.torrent` file using the `bencode` library. This file contains metadata about the content to be downloaded, including the file's info hash, the list of pieces, and tracker URLs

### 2. Tracker communication

Tracker communication is performed via HTTP  GET requests, which include parameters such as `peer_id`, `info_hash`, `port`, and `left`. The tracker responds a list of peers that have the file or part of it.

### 3. Peer communication

After receiving the list of peers from the tracker, the client established TCP connections with them. The process is handled in `peer.go` and includes the following steps:

#### handshake

The handshake is the initial message exchanged between peers, ensureing they are working on the same torrent by exchanging `infoSHA` and `peerId`.

The message includes:

- 1 byte: length of the protocol
- 19 bytes: protocol indentifier(`BitTorrent`)
- 8 bytes: reserved for scalability (set to 0x00)
- 20 bytes: `InfoSHA` (SHA-1 hash of the torrent's info dictionary)
- 20 bytes: `peerId` (unique identifier of the client)

#### piece exchange

The client requests specific pieces from peers, based on the peer's bitfield, which indicates the pieces the peer has available, downloading them and verifying their integrity using the piece SHA-1 hashes.

The messages exchanged between peers include a message type (e.g., choke, unchoke, interested, not interested) and content (e.g., bitfield, piece data). The default state is `choke`, which prevents uploading until the peer is unchoked.

Peer connection details include:

- **Address**: the client connects to peers using the IP address and port obtained from the tracker.
- **Handshake**: the handshake process is critical for establishing a connection and exchaning basic information
- **Peer connection**: after a successful handshake, a new peer connection is established, and the client begins downloading pieces.

## File structure

### 1. `bencode` Directory

* **Purpose** : Handles encoding and decoding of data in the bencode format, which is used for reading and writing `.torrent` files and tracker responses.
* **Files** :
* `bencode.go`: Core logic for bencode encoding and decoding.
* `marshal.go`: Implements the serialization (marshaling) of Go data structures into bencode format.
* `parser.go`: Implements the deserialization (unmarshaling) of bencode data into Go structures.

### 2. `torrent` Directory

This directory contains the core files responsible for parsing the torrent file, communicating with the tracker, managing peer connections, and downloading the file pieces.

#### a. `torrent_file.go`

* **Purpose** : Parses the `.torrent` file and extracts metadata necessary for downloading the content.
* **Key Functions** :
* `ParseFile`: Reads and parses the torrent file, extracts the announce URL, file name, file length, piece length, and computes the SHA-1 hashes of the file's pieces.

#### b. `tracker.go`

* **Purpose** : Handles communication with the tracker to retrieve a list of peers.
* **Key Functions** :
* `ContactTracker`: Sends an HTTP GET request to the tracker's announce URL with the required parameters and processes the response to extract the peer list.

#### c. `peer.go`

* **Purpose** : Manages the connection to peers, including the handshake process and piece requests.
* **Key Functions** :
* `handshake`: Establishes the initial connection with a peer, ensuring both parties are working on the same torrent.
* `fillBitfield`: Retrieves the bitfield from a peer, indicating which pieces the peer has available.
* `ReadMsg` and `WriteMsg`: Manage the reading and writing of messages between peers.

#### d. `download.go`

* **Purpose** : Manages the overall downloading process by coordinating tasks among multiple peers.
* **Key Functions** :
* `Download`: Manages the download process, splitting the torrent into piece tasks and coordinating peer routines to download each piece concurrently.
* `peerRoutine`: Handles communication with a single peer, requesting and downloading pieces, and verifying their integrity.

# go-torrent

tracker: http protocol, GET request, url+params

    peer id: clients

    info hash: file

    port

    left

response(bencode) --unmarshal-->

-->ip:port of peers

peer.go

1. handshake: bit torrent protocol, **peerId, InfoSHA**

   message 5 parts

   1. byte: length of 2nd part(0x13)
   2. bit torrent protocol 19bytes
   3. 0x00 (8) for scalability
   4. infoSHA
   5. my peerId
2. what pieces: bit field(bit map)
3. (peerMsg.go) kind(9) +contents(bit map), default kind: choke(i downloaded piece1 but not upload; unchoke upload automatically)
4. download specific pieces

peer connection(peer conn)

1. addr
2. handshake
3. new a peer conn already handshook

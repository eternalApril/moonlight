package server

import (
	"net"
	"sync"

	"github.com/eternalApril/moonlight/internal/resp"
)

type Peer struct {
	conn   net.Conn
	reader *resp.Decoder
	writer *resp.Encoder
	mu     sync.Mutex
}

func NewPeer(conn net.Conn) *Peer {
	return &Peer{
		conn:   conn,
		reader: resp.NewDecoder(conn),
		writer: resp.NewEncoder(conn),
	}
}

func (p *Peer) Send(v resp.Value) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.writer.Write(v)
}

func (p *Peer) ReadCommand() (resp.Value, error) {
	return p.reader.Read()
}

func (p *Peer) Close() error {
	return p.conn.Close()
}

package server

import (
	"net"

	"github.com/eternalApril/moonlight/internal/resp"
)

type Peer struct {
	conn   net.Conn
	reader *resp.Decoder
	writer *resp.Encoder
}

func NewPeer(conn net.Conn) *Peer {
	return &Peer{
		conn:   conn,
		reader: resp.NewDecoder(conn),
		writer: resp.NewEncoder(conn),
	}
}

func (p *Peer) Send(v resp.Value) error {
	return p.writer.Write(v)
}

func (p *Peer) ReadCommand() (resp.Value, error) {
	return p.reader.Read()
}

func (p *Peer) Close() error {
	return p.conn.Close()
}

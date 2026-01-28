package server

import (
	"net"
	"sync"

	"github.com/eternalApril/moonlight/internal/resp"
)

// Peer represents a connected client.
// It wraps a network connection and provides synchronized methods for reading and writing RESP-encoded data
type Peer struct {
	conn   net.Conn
	reader *resp.Decoder
	writer *resp.Encoder
	mu     sync.Mutex
}

// NewPeer initializes a new client peer from a network connection
func NewPeer(conn net.Conn) *Peer {
	return &Peer{
		conn:   conn,
		reader: resp.NewDecoder(conn),
		writer: resp.NewEncoder(conn),
	}
}

// Send encodes and writes a RESP value to the client.
// This method is thread-safe and can be called from multiple goroutines
func (p *Peer) Send(v resp.Value) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.writer.Write(v)
}

// ReadCommand reads and decodes the next RESP value from the client's input stream
func (p *Peer) ReadCommand() (resp.Value, error) {
	return p.reader.Read()
}

// Close terminates the underlying network connection
func (p *Peer) Close() error {
	return p.conn.Close()
}

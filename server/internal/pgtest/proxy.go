package pgtest

import (
	"errors"
	"io"
	"net"
	"sync"
	"testing"
)

// Proxy forwards TCP connections to a database so a test can make the
// database unreachable on demand.
type Proxy struct {
	ln     net.Listener
	target string

	mu    sync.Mutex
	conns []net.Conn
	cut   bool
}

// NewProxy listens on a free local port and forwards each connection to
// target ("host:port"). It is cut when the test ends.
func NewProxy(t testing.TB, target string) *Proxy {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("proxy listen: %v", err)
	}
	p := &Proxy{ln: ln, target: target}
	go p.accept()
	t.Cleanup(p.Cut)
	return p
}

// Addr is the "host:port" clients connect to.
func (p *Proxy) Addr() string { return p.ln.Addr().String() }

// Cut closes every open connection and refuses new ones.
func (p *Proxy) Cut() {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.cut {
		return
	}
	p.cut = true
	_ = p.ln.Close()
	for _, c := range p.conns {
		_ = c.Close()
	}
}

func (p *Proxy) accept() {
	for {
		client, err := p.ln.Accept()
		if errors.Is(err, net.ErrClosed) {
			return
		}
		if err != nil {
			continue
		}
		server, err := net.Dial("tcp", p.target)
		if err != nil {
			_ = client.Close()
			continue
		}
		if !p.track(client, server) {
			return
		}
		go pipe(client, server)
		go pipe(server, client)
	}
}

// track records the pair so Cut can close it; it reports false, closing the
// pair, if the proxy was cut in the meantime.
func (p *Proxy) track(conns ...net.Conn) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.cut {
		for _, c := range conns {
			_ = c.Close()
		}
		return false
	}
	p.conns = append(p.conns, conns...)
	return true
}

func pipe(dst, src net.Conn) {
	_, _ = io.Copy(dst, src)
	_ = dst.Close()
	_ = src.Close()
}

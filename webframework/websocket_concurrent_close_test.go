package gw_web

import (
	"bufio"
	"crypto/sha1"
	"encoding/base64"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"sync/atomic"
	"testing"
	"time"

	"github.com/fasthttp/websocket"
	contribws "github.com/gofiber/contrib/v3/websocket"
)

// drainingConn は contrib の Close の約束を真似る: 1 回目は送り切ってから閉じ、送り切る間に来た 2 回目はすぐ返る
type drainingConn struct {
	net.Conn
	entered chan struct{}
	release chan struct{}
	started atomic.Bool
	flushed atomic.Bool
	raw     *observedRaw
}

type observedRaw struct {
	net.Conn
	owner *drainingConn
	early atomic.Bool
}

func (r *observedRaw) Close() error {
	if !r.owner.flushed.Load() {
		r.early.Store(true)
	}
	return r.Conn.Close()
}

func (c *drainingConn) Close() error {
	if c.started.CompareAndSwap(false, true) {
		close(c.entered)
		<-c.release
		c.flushed.Store(true)
		return c.raw.Close()
	}
	return nil
}

func (c *drainingConn) UnsafeConn() net.Conn { return c.raw }

// handler の Close が送り切る途中に、shutdown の Close が重なっても、下の通信路を先に閉じない
// (Codex のレビューでの再現をもとにした試験)
func TestConcurrentCloseWaitsForTheOngoingDrain(t *testing.T) {
	a, b := net.Pipe()
	defer b.Close()
	transport := &drainingConn{Conn: a, entered: make(chan struct{}), release: make(chan struct{})}
	transport.raw = &observedRaw{Conn: a, owner: transport}
	go func() {
		req, err := http.ReadRequest(bufio.NewReader(b))
		if err != nil {
			return
		}
		hash := sha1.Sum([]byte(req.Header.Get("Sec-WebSocket-Key") + "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"))
		fmt.Fprintf(b, "HTTP/1.1 101 Switching Protocols\r\nUpgrade: websocket\r\nConnection: Upgrade\r\nSec-WebSocket-Accept: %s\r\n\r\n", base64.StdEncoding.EncodeToString(hash[:]))
	}()
	u, _ := url.Parse("ws://localhost/ws")
	socket, _, err := websocket.NewClient(transport, u, nil, 1024, 1024)
	if err != nil {
		t.Fatal(err)
	}
	conn := &WebSocketConn{Conn: &contribws.Conn{Conn: socket}}

	firstDone := make(chan struct{})
	go func() { _ = conn.Close(); close(firstDone) }()
	select {
	case <-transport.entered:
	case <-time.After(time.Second):
		t.Fatal("first Close did not start")
	}
	secondDone := make(chan struct{})
	go func() { _ = conn.Close(); close(secondDone) }()
	select {
	case <-secondDone:
		t.Fatal("second Close must wait for the first Close to finish draining")
	case <-time.After(50 * time.Millisecond):
	}
	close(transport.release)
	<-firstDone
	<-secondDone
	if transport.raw.early.Load() {
		t.Fatal("second Close closed the raw socket before the first Close finished draining")
	}
}

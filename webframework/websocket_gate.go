package gw_web

import (
	"context"
	"net/http"
	"sync"
)

// webSocketGateはWebAppのshutdown後のupgradeを拒否し、開始済みの接続をcloseします。
type webSocketGate struct {
	mu          sync.Mutex
	accepting   bool
	connections map[*WebSocketConn]struct{}
	drained     chan struct{}
	drainOnce   sync.Once
}

func newWebSocketGate() *webSocketGate {
	return &webSocketGate{
		accepting:   true,
		connections: make(map[*WebSocketConn]struct{}),
		drained:     make(chan struct{}),
	}
}

func (g *webSocketGate) middleware(c *WebCtx) error {
	if g == nil || !g.isAccepting() {
		return c.Status(http.StatusServiceUnavailable).SendString("server is shutting down")
	}
	return c.Next()
}

// wrapはupgrade判定直後のshutdown競合も含め、接続を自動的に追跡します。
func (g *webSocketGate) wrap(handler WsHandler) WsHandler {
	return func(conn *WebSocketConn) {
		if g == nil || !g.add(conn) {
			_ = conn.Close()
			return
		}
		defer g.remove(conn)
		handler(conn)
	}
}

func (g *webSocketGate) isAccepting() bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.accepting
}

func (g *webSocketGate) add(conn *WebSocketConn) bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	if !g.accepting {
		return false
	}
	g.connections[conn] = struct{}{}
	return true
}

func (g *webSocketGate) remove(conn *WebSocketConn) {
	g.mu.Lock()
	defer g.mu.Unlock()
	delete(g.connections, conn)
	g.markDrainedIfEmpty()
}

func (g *webSocketGate) closeAll() {
	if g == nil {
		return
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	g.accepting = false
	// removeと同じlock内でcloseし、handler終了後にpoolへ返された接続を触らないようにします。
	for conn := range g.connections {
		_ = conn.Close()
	}
	g.markDrainedIfEmpty()
}

// waitは追跡中のWebSocket handlerがすべて終了するまで待機します。
func (g *webSocketGate) wait(ctx context.Context) error {
	if g == nil {
		return nil
	}
	select {
	case <-g.drained:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// markDrainedIfEmptyはg.muを保持した状態で呼び出します。
func (g *webSocketGate) markDrainedIfEmpty() {
	if !g.accepting && len(g.connections) == 0 {
		g.drainOnce.Do(func() { close(g.drained) })
	}
}

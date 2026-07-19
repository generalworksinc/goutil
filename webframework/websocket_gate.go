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
	connections map[*trackedWebSocket]struct{}
	drained     chan struct{}
	drainOnce   sync.Once
}

// trackedWebSocketはcloseとhandler終了を直列化し、frameworkのconnection pool返却と競合させません。
type trackedWebSocket struct {
	mu   sync.Mutex
	conn *WebSocketConn
}

func newWebSocketGate() *webSocketGate {
	return &webSocketGate{
		accepting:   true,
		connections: make(map[*trackedWebSocket]struct{}),
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
		tracked := &trackedWebSocket{conn: conn}
		if g == nil || !g.add(tracked) {
			_ = conn.Close()
			return
		}
		defer func() {
			tracked.mu.Lock()
			tracked.conn = nil
			tracked.mu.Unlock()
			g.remove(tracked)
		}()
		handler(conn)
	}
}

func (g *webSocketGate) isAccepting() bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.accepting
}

func (g *webSocketGate) add(conn *trackedWebSocket) bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	if !g.accepting {
		return false
	}
	g.connections[conn] = struct{}{}
	return true
}

func (g *webSocketGate) remove(conn *trackedWebSocket) {
	g.mu.Lock()
	delete(g.connections, conn)
	g.closeDrainedIfEmpty()
	g.mu.Unlock()
}

func (g *webSocketGate) closeAll() {
	if g == nil {
		return
	}
	g.mu.Lock()
	g.accepting = false
	connections := make([]*trackedWebSocket, 0, len(g.connections))
	for conn := range g.connections {
		connections = append(connections, conn)
	}
	g.closeDrainedIfEmpty()
	g.mu.Unlock()

	for _, tracked := range connections {
		tracked.mu.Lock()
		if tracked.conn != nil {
			_ = tracked.conn.Close()
		}
		tracked.mu.Unlock()
	}
}

// waitは追跡中のWebSocket handlerがすべて終了するまで待機します。
func (g *webSocketGate) wait(ctx context.Context) error {
	if g == nil {
		return nil
	}
	if ctx == nil {
		<-g.drained
		return nil
	}
	select {
	case <-g.drained:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// closeDrainedIfEmptyはg.muを保持した状態で呼び出します。
func (g *webSocketGate) closeDrainedIfEmpty() {
	if !g.accepting && len(g.connections) == 0 {
		g.drainOnce.Do(func() { close(g.drained) })
	}
}

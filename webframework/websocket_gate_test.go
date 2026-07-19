package gw_web

import (
	"context"
	"errors"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/fasthttp/websocket"
	"github.com/gofiber/fiber/v3"
)

func TestWebSocketRoutesAutomaticallyTrackAndRejectConnections(t *testing.T) {
	tests := []struct {
		name     string
		register func(*WebApp, WsHandler)
	}{
		{
			name: "WebApp.WsGet",
			register: func(app *WebApp, handler WsHandler) {
				app.WsGet("/ws", handler)
			},
		},
		{
			name: "WebApp.WsGetWithConfig",
			register: func(app *WebApp, handler WsHandler) {
				app.WsGetWithConfig("/ws", WebSocketConfig{}, handler)
			},
		},
		{
			name: "WebGroup.WsGet",
			register: func(app *WebApp, handler WsHandler) {
				app.Group("/group").WsGet("/ws", handler)
			},
		},
		{
			name: "WebGroup.WsGetWithConfig",
			register: func(app *WebApp, handler WsHandler) {
				app.Group("/group").WsGetWithConfig("/ws", WebSocketConfig{}, handler)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := newWebSocketGateTestApp()
			connected := make(chan struct{})
			disconnected := make(chan struct{})
			tt.register(app, func(conn *WebSocketConn) {
				defer close(disconnected)
				close(connected)
				_, _, _ = conn.ReadMessage()
			})

			path := "/ws"
			if tt.name == "WebGroup.WsGet" || tt.name == "WebGroup.WsGetWithConfig" {
				path = "/group/ws"
			}
			address := startWebSocketGateTestServer(t, app)
			conn := dialWebSocketGateTest(t, "ws://"+address+path)
			defer conn.Close()
			select {
			case <-connected:
			case <-time.After(2 * time.Second):
				t.Fatal("WebSocket handler did not start")
			}
			app.webSockets.mu.Lock()
			trackedCount := len(app.webSockets.connections)
			app.webSockets.mu.Unlock()
			if trackedCount != 1 {
				t.Fatalf("tracked WebSocket connections=%d, want=1", trackedCount)
			}

			app.webSockets.closeAll()
			select {
			case <-disconnected:
			case <-time.After(2 * time.Second):
				t.Fatal("WebSocket handler did not stop after gate close")
			}
			assertWebSocketGateConnectionClosed(t, conn)

			newConn, response, err := websocket.DefaultDialer.Dial("ws://"+address+path, nil)
			if newConn != nil {
				_ = newConn.Close()
			}
			if response != nil && response.Body != nil {
				defer response.Body.Close()
			}
			if err == nil {
				t.Fatal("WebSocket upgrade must be rejected after gate close")
			}
			if response == nil || response.StatusCode != http.StatusServiceUnavailable {
				status := 0
				if response != nil {
					status = response.StatusCode
				}
				t.Fatalf("status=%d, want=%d", status, http.StatusServiceUnavailable)
			}
		})
	}
}

func TestWebAppShutdownMethodsCloseTrackedWebSockets(t *testing.T) {
	tests := []struct {
		name     string
		shutdown func(*WebApp) error
	}{
		{name: "Shutdown", shutdown: func(app *WebApp) error { return app.Shutdown() }},
		{name: "ShutdownWithTimeout", shutdown: func(app *WebApp) error { return app.ShutdownWithTimeout(2 * time.Second) }},
		{name: "ShutdownWithContext", shutdown: func(app *WebApp) error {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			return app.ShutdownWithContext(ctx)
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := newWebSocketGateTestApp()
			connected := make(chan struct{})
			disconnected := make(chan struct{})
			app.WsGet("/ws", func(conn *WebSocketConn) {
				defer close(disconnected)
				close(connected)
				_, _, _ = conn.ReadMessage()
			})
			address := startWebSocketGateTestServer(t, app)
			conn := dialWebSocketGateTest(t, "ws://"+address+"/ws")
			defer conn.Close()
			select {
			case <-connected:
			case <-time.After(2 * time.Second):
				t.Fatal("WebSocket handler did not start")
			}

			if err := tt.shutdown(app); err != nil {
				t.Fatal(err)
			}
			select {
			case <-disconnected:
			default:
				t.Fatal("WebSocket handler must stop before shutdown returns")
			}
			assertWebSocketGateConnectionClosed(t, conn)
		})
	}
}

func TestShutdownWithContextStopsWaitingAtDeadline(t *testing.T) {
	app := newWebSocketGateTestApp()
	connected := make(chan struct{})
	releaseHandler := make(chan struct{})
	handlerStopped := make(chan struct{})
	app.WsGet("/ws", func(*WebSocketConn) {
		close(connected)
		<-releaseHandler
		close(handlerStopped)
	})
	address := startWebSocketGateTestServer(t, app)
	conn := dialWebSocketGateTest(t, "ws://"+address+"/ws")
	defer conn.Close()
	select {
	case <-connected:
	case <-time.After(2 * time.Second):
		t.Fatal("WebSocket handler did not start")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	if err := app.ShutdownWithContext(ctx); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("shutdown error=%v, want=%v", err, context.DeadlineExceeded)
	}
	close(releaseHandler)
	select {
	case <-handlerStopped:
	case <-time.After(2 * time.Second):
		t.Fatal("WebSocket handler did not stop after release")
	}
}

func assertWebSocketGateConnectionClosed(t *testing.T, conn *websocket.Conn) {
	t.Helper()
	_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, _, err := conn.ReadMessage()
	if err == nil {
		t.Fatal("WebSocket connection must be closed")
	}
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		t.Fatalf("WebSocket connection remained open until read timeout: %v", err)
	}
}

func newWebSocketGateTestApp() *WebApp {
	return NewApp(func(ctx *WebCtx, err error) error {
		return ctx.Status(http.StatusInternalServerError).SendString(err.Error())
	})
}

func startWebSocketGateTestServer(t *testing.T, app *WebApp) string {
	t.Helper()
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	go func() {
		_ = app.App.(*fiber.App).Listener(listener, fiber.ListenConfig{DisableStartupMessage: true})
	}()
	t.Cleanup(func() {
		_ = app.ShutdownWithTimeout(time.Second)
		_ = listener.Close()
	})
	return listener.Addr().String()
}

func dialWebSocketGateTest(t *testing.T, url string) *websocket.Conn {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		conn, response, err := websocket.DefaultDialer.Dial(url, nil)
		if response != nil && response.Body != nil {
			_ = response.Body.Close()
		}
		if err == nil {
			return conn
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("could not connect WebSocket: %s", url)
	return nil
}

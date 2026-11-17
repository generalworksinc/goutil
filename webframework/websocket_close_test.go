package gw_web

import (
	"testing"
	"time"

	"github.com/fasthttp/websocket"
)

// 返事を書いてすぐ Close しても、返事が相手に届く。contrib の websocket は返事をためてまとめて送るので、
// Close がその部品を通らずに下の通信路を閉じると、ためた返事が捨てられる(受け取った文が 12 バイト以上のとき)
func TestWebSocketCloseDeliversWhatWasJustWritten(t *testing.T) {
	for _, first := range []string{"short", `{"type":"heartbeat","sentAt":"2026-11-17T00:00:00Z"}`} {
		app := newWebSocketGateTestApp()
		app.WsGet("/ws", func(conn *WebSocketConn) {
			_, _, _ = conn.ReadMessage()
			if err := conn.WriteMessageText([]byte("error: first message must be hello")); err != nil {
				t.Errorf("server write: %v", err)
			}
			_ = conn.Close()
		})
		address := startWebSocketGateTestServer(t, app)
		conn := dialWebSocketGateTest(t, "ws://"+address+"/ws")
		if err := conn.WriteMessage(websocket.TextMessage, []byte(first)); err != nil {
			t.Fatal(err)
		}
		_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
		_, msg, err := conn.ReadMessage()
		if err != nil || string(msg) != "error: first message must be hello" {
			t.Fatalf("first=%q: the reply written before Close must arrive: msg=%q err=%v", first, msg, err)
		}
		_ = conn.Close()
	}
}

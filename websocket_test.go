package websocket

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/gorilla/websocket"
	"github.com/lunny/tango"
)

var (
	host     string = "http://localhost:8000"
	endpoint string = "ws://localhost:8000"
)

type WebSocketAction struct {
	tango.Ctx
	Message
}

func (w *WebSocketAction) Get() {
	if w.IsUpgrade() {
		fmt.Println("websocket request")

		w.OnConnected(func(ws *websocket.Conn, sender chan []byte) {
			fmt.Println("on connected")
		}).OnReceived(func(ws *websocket.Conn, data []byte) {
			fmt.Println("on received", string(data))
		}).OnClosed(func(ws *websocket.Conn) {
			fmt.Println("ws closed")
			done <- true
		}).ListenAndServe()
	} else {
		fmt.Println("non-websocket request")
	}
}

var done = make(chan bool)

func TestWebSocket(t *testing.T) {
	buff := bytes.NewBufferString("")
	recorder := httptest.NewRecorder()
	recorder.Body = buff

	tg := tango.Classic()
	tg.Use(New())
	tg.Get("/", new(WebSocketAction))
	go tg.Run()

	ws, _ := connectSocket(t, "/")
	err := ws.WriteMessage(websocket.TextMessage, []byte("hhhh"))
	if err != nil {
		t.Errorf("Writing to the socket failed with %s", err.Error())
	}

	ws.Close()

	expect(t, err, nil)

	<-done
}

func connectSocket(t *testing.T, path string) (*websocket.Conn, *http.Response) {
	header := make(http.Header)
	header.Add("Origin", host)
	ws, resp, err := websocket.DefaultDialer.Dial(endpoint+path, header)
	if err != nil {
		t.Fatalf("Connecting the socket failed: %s", err.Error())
	}

	return ws, resp
}

/* Test Helpers */
func expect(t *testing.T, a interface{}, b interface{}) {
	if a != b {
		t.Errorf("Expected %v (type %v) - Got %v (type %v)", b, reflect.TypeOf(b), a, reflect.TypeOf(a))
	}
}

func refute(t *testing.T, a interface{}, b interface{}) {
	if a == b {
		t.Errorf("Did not expect %v (type %v) - Got %v (type %v)", b, reflect.TypeOf(b), a, reflect.TypeOf(a))
	}
}

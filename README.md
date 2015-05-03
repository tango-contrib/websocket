# websocket middleware for Tango

** IN DEVELOPMENT, API WILL CHANGED FREQUENTLY **

```Go

type Action struct {
    websocket.Message
}

func (w *Action) Get() {
    w.OnConnected(func(ws *websocket.Conn, sender chan []byte) {
        fmt.Println("on connected")
    }).OnReceived(func(ws *websocket.Conn, data []byte) {
        fmt.Println("on received", string(data))
    }).OnClosed(func(ws *websocket.Conn) {
        fmt.Println("ws closed")
    }).ListenAndServe()
}

func main() {
    tg := tango.Classic()
    tg.Use(websocket.New())
    tg.Get("/", new(Action))
    tg.Run()
}
```
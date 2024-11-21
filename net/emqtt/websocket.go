package emqtt

import (
	"github.com/soedev/soelib/common/soelog"
	"golang.org/x/net/websocket"
	"io"
	"net"
	"net/http"
	"net/url"
)

// DefaultListenAndServeWebsocket DefaultListenAndServeWebsocket
func DefaultListenAndServeWebsocket() error {
	if err := AddWebsocketHandler("/mqtt", "test.mosquitto.org:1883"); err != nil {
		return err
	}
	return ListenAndServeWebsocket(":1234")
}

// AddWebsocketHandler AddWebsocketHandler
func AddWebsocketHandler(urlPattern string, uri string) error {

	u, err := url.Parse(uri)
	if err != nil {
		soelog.Logger.Error("surgemq/main: " + err.Error())
		return err
	}

	h := func(ws *websocket.Conn) {
		err := WebsocketTcpProxy(ws, u.Scheme, u.Host)
		if err != nil {
			return
		}
	}
	http.Handle(urlPattern, websocket.Handler(h))
	return nil
}

// ListenAndServeWebsocket start a listener that proxies websocket <-> tcp
func ListenAndServeWebsocket(addr string) error {
	return http.ListenAndServe(addr, nil)
}

// ListenAndServeWebsocketSecure starts an HTTPS listener
func ListenAndServeWebsocketSecure(addr string, cert string, key string) error {
	return http.ListenAndServeTLS(addr, cert, key, nil)
}

// ioCopyWs copy from websocket to writer, this copies the binary frames as is
func ioCopyWs(src *websocket.Conn, dst io.Writer) (int, error) {
	var buffer []byte
	count := 0
	for {
		err := websocket.Message.Receive(src, &buffer)
		if err != nil {
			return count, err
		}
		n := len(buffer)
		count += n
		i, err := dst.Write(buffer)
		if err != nil || i < 1 {
			return count, err
		}
	}
}

// ioWsCopy copy from reader to websocket, this copies the binary frames as is
func ioWsCopy(src io.Reader, dst *websocket.Conn) (int, error) {
	buffer := make([]byte, 2048)
	count := 0
	for {
		n, err := src.Read(buffer)
		if err != nil || n < 1 {
			return count, err
		}
		count += n
		err = websocket.Message.Send(dst, buffer[0:n])
		if err != nil {
			return count, err
		}
	}
}

// WebsocketTcpProxy handler that proxies websocket <-> unix domain socket
func WebsocketTcpProxy(ws *websocket.Conn, nettype string, host string) error {
	client, err := net.Dial(nettype, host)
	if err != nil {
		return err
	}
	defer func(client net.Conn) {
		_ = client.Close()
	}(client)
	defer func(ws *websocket.Conn) {
		_ = ws.Close()
	}(ws)
	chDone := make(chan bool)

	go func() {
		_, _ = ioWsCopy(client, ws)
		chDone <- true
	}()
	go func() {
		_, _ = ioCopyWs(ws, client)
		chDone <- true
	}()
	<-chDone
	return nil
}

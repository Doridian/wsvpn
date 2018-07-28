package wstun_shared

import (
	"fmt"
	"github.com/gorilla/websocket"
	"github.com/songgao/water"
	"log"
	"strings"
	"sync"
	"time"
)

type CommandHandler interface {
	HandleCommand(args []string) error
}

func SendCommand(conn *websocket.Conn, writeLock *sync.Mutex, command string, args ...string) error {
	data := fmt.Sprintf("%s|%s", command, strings.Join(args, "|"))
	writeLock.Lock()
	err := conn.WriteMessage(websocket.TextMessage, []byte(data))
	writeLock.Unlock()
	return err
}

func HandleSocket(iface *water.Interface, conn *websocket.Conn, writeLock *sync.Mutex,
	wg *sync.WaitGroup, handlers map[string]CommandHandler) {

	wg.Add(1)
	go func() {
		defer wg.Done()
		defer conn.Close()

		packet := make([]byte, 2000)

		for {
			n, err := iface.Read(packet)
			if err != nil {
				log.Println(err)
				break
			}
			writeLock.Lock()
			err = conn.WriteMessage(websocket.BinaryMessage, packet[:n])
			writeLock.Unlock()
			if err != nil {
				break
			}
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		defer conn.Close()

		for {
			msgType, msg, err := conn.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway) {
					log.Println(err)
				}
				break
			}
			if msgType == websocket.BinaryMessage {
				iface.Write(msg)
			} else if msgType == websocket.TextMessage {
				str := strings.Split(string(msg), "|")
				handler := handlers[str[0]]
				if handler == nil {
					log.Printf("Unknown in-band command %s", str[0])
					continue
				}
				err = handler.HandleCommand(str[1:])
				if err != nil {
					log.Printf("Error in in-band command %s: %v", str[0], err)
				}
			}
		}
	}()

	keepAlive(conn, writeLock, wg)

	wg.Wait()
}

func keepAlive(c *websocket.Conn, l *sync.Mutex, wg *sync.WaitGroup) {
	timeout := time.Duration(30) * time.Second

	lastResponse := time.Now()
	c.SetPongHandler(func(msg string) error {
		lastResponse = time.Now()
		return nil
	})

	wg.Add(1)
	go func() {
		defer wg.Done()
		defer c.Close()

		for {
			l.Lock()
			err := c.WriteMessage(websocket.PingMessage, []byte("keepalive"))
			l.Unlock()
			if err != nil {
				return
			}
			time.Sleep(timeout / 2)
			if time.Now().Sub(lastResponse) > timeout {
				return
			}
		}
	}()
}

package wstun_shared

import (
	"errors"
	"fmt"
	"github.com/gorilla/websocket"
	"github.com/songgao/water"
	"log"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

var lastCommandId uint64 = 0

type CommandHandler func(args []string) error

func RawSendCommand(conn *websocket.Conn, writeLock *sync.Mutex, commandId string, command string, args ...string) error {
	data := []byte(fmt.Sprintf("%s|%s|%s", commandId, command, strings.Join(args, "|")))
	writeLock.Lock()
	err := conn.WriteMessage(websocket.TextMessage, data)
	writeLock.Unlock()
	return err
}

func SendCommand(conn *websocket.Conn, writeLock *sync.Mutex, command string, args ...string) error {
	return RawSendCommand(conn, writeLock, fmt.Sprintf("%d", atomic.AddUint64(&lastCommandId, 1)), command, args...)
}

func HandleSocket(connId string, iface *water.Interface, conn *websocket.Conn, writeLock *sync.Mutex,
	wg *sync.WaitGroup, handlers map[string]CommandHandler) {

	wg.Add(1)
	go func() {
		defer wg.Done()
		defer conn.Close()
		defer iface.Close()

		packet := make([]byte, 2000)

		for {
			n, err := iface.Read(packet)
			if err != nil {
				log.Printf("[%s] Error reading packet from tun: %v", connId, err)
				break
			}
			writeLock.Lock()
			err = conn.WriteMessage(websocket.BinaryMessage, packet[:n])
			writeLock.Unlock()
			if err != nil {
				log.Printf("[%s] Error writing packet to WS: %v", connId, err)
				break
			}
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		defer conn.Close()
		defer iface.Close()

		for {
			msgType, msg, err := conn.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway) {
					log.Printf("[%s] Error reading packet from WS: %v", connId, err)
				}
				break
			}
			if msgType == websocket.BinaryMessage {
				iface.Write(msg)
			} else if msgType == websocket.TextMessage {
				str := strings.Split(string(msg), "|")
				if len(str) < 2 {
					log.Printf("[%s] Invalid in-band command structure", connId)
					continue
				}

				commandId := str[0]
				commandName := str[1]
				if commandName == "reply" {
					commandResult := "N/A"
					if len(str) > 2 {
						commandResult = str[2]
					}
					log.Printf("[%s] Got command reply ID %s: %s", connId, commandId, commandResult)
					continue
				}

				handler := handlers[commandName]
				if handler == nil {
					err = errors.New("Unknown command")
				} else {
					err = handler(str[2:])
				}
				if err != nil {
					log.Printf("[%s] Error in in-band command %s: %v", connId, commandName, err)
				}

				RawSendCommand(conn, writeLock, commandId, "reply", fmt.Sprintf("%v", err == nil))
			}
		}
	}()

	timeout := time.Duration(30) * time.Second

	lastResponse := time.Now()
	conn.SetPongHandler(func(msg string) error {
		lastResponse = time.Now()
		return nil
	})

	wg.Add(1)
	go func() {
		defer wg.Done()
		defer conn.Close()
		defer iface.Close()

		for {
			writeLock.Lock()
			err := conn.WriteMessage(websocket.PingMessage, []byte{})
			writeLock.Unlock()
			if err != nil {
				log.Printf("[%s] Error writing ping frame: %v", connId, err)
				break
			}
			time.Sleep(timeout / 2)
			if time.Now().Sub(lastResponse) > timeout {
				log.Printf("[%s] Ping timeout", connId)
				break
			}
		}
	}()
}

package service

import (
	"encoding/json"
	"fmt"
	"tool"

	"github.com/gorilla/websocket"
)

// ClientManager is a websocket manager
type ClientManager struct {
	Clients    map[*Client]bool
	Broadcast  chan []byte
	Register   chan *Client
	Unregister chan *Client
}

// Client is a websocket client
type Client struct {
	ID           string
	Subscription int
	Socket       *websocket.Conn
	Send         chan []byte
}

// Message is an object for websocket message which is mapped to json type
type Message struct {
	MType   int
	Sender  string
	Content string
}

// the type of Message
const (
	txt = iota
	emoji
	img
)

// Manager define a ws server manager
var Manager = ClientManager{
	Broadcast:  make(chan []byte),
	Register:   make(chan *Client),
	Unregister: make(chan *Client),
	Clients:    make(map[*Client]bool),
}

// Channel is what Clients have subscripted
var Channel = make(map[int]map[*Client]bool)

// Start is to start a ws server
func (manager *ClientManager) Start() {
	for {
		select {
		case conn := <-manager.Register:
			manager.Clients[conn] = true
			if _, ok := Channel[conn.Subscription]; !ok {
				Channel[conn.Subscription] = make(map[*Client]bool)
				fmt.Println("channel-", conn.Subscription, ":Established")
			}
			Channel[conn.Subscription][conn] = true
			jsonMessage, _ := json.Marshal(&Message{Content: "/A new socket has connected."})
			manager.Send(jsonMessage, conn.Subscription, conn)
		case conn := <-manager.Unregister:
			if _, ok := manager.Clients[conn]; ok {
				close(conn.Send)
				delete(Channel[conn.Subscription], conn)
				if len(Channel[conn.Subscription]) == 0 {
					delete(Channel, conn.Subscription)
					fmt.Println("channel-", conn.Subscription, ":Relieved")
				}
				delete(manager.Clients, conn)
				jsonMessage, _ := json.Marshal(&Message{Content: "/A socket has disconnected."})
				manager.Send(jsonMessage, conn.Subscription, conn)
			}
		case message := <-manager.Broadcast:
			for conn := range manager.Clients {
				select {
				case conn.Send <- message:
				default:
					close(conn.Send)
					delete(manager.Clients, conn)
				}
			}
		}
	}
}

// Send is to send ws message to ws client
func (manager *ClientManager) Send(message []byte, to int, ignore *Client) {
	for conn := range Channel[to] {
		if conn != ignore {
			conn.Send <- message
		}
	}
}

func (c *Client) Read() {
	defer func() {
		Manager.Unregister <- c
		c.Socket.Close()
	}()

	for {
		_, message, err := c.Socket.ReadMessage()
		if err != nil {
			Manager.Unregister <- c
			c.Socket.Close()
			break
		}
		sMap := tool.GetMap()
		if _, ok := sMap.CheckSensitive(string(message)); ok {
			jsonMessage, _ := json.Marshal(&Message{Content: "含有敏感词!"})
			c.Send <- jsonMessage
			continue
		}
		jsonMessage, _ := json.Marshal(&Message{Sender: c.ID, Content: string(message)})
		for conn := range Channel[c.Subscription] {
			conn.Send <- jsonMessage
		}
	}
}

func (c *Client) Write() {
	defer func() {
		c.Socket.Close()
	}()

	for {
		select {
		case message, ok := <-c.Send:
			if !ok {
				c.Socket.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			c.Socket.WriteMessage(websocket.TextMessage, message)
		}
	}
}

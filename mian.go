package main

import (
	"math/rand"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	uuid "github.com/satori/go.uuid"

	"service"
)

// WsPage is a websocket handler
func WsPage(c *gin.Context) {
	// change the reqest to websocket model
	conn, error := (&websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}).Upgrade(c.Writer, c.Request, nil)
	if error != nil {
		http.NotFound(c.Writer, c.Request)
		return
	}
	// websocket connect
	id, _ := uuid.NewV4()
	rand.Seed(time.Now().Unix())
	sid := rand.Intn(3)
	client := &service.Client{ID: id.String(), Subscription: sid, Socket: conn, Send: make(chan []byte)}

	service.Manager.Register <- client

	go client.Read()
	go client.Write()
}

func main() {
	go service.Manager.Start()

	r := gin.Default()
	r.GET("/wsgo", WsPage)
	r.Run() // listen and serve on 0.0.0.0:8080
}

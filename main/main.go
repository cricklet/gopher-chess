package main

import (
	"bufio"
	"fmt"
	"net/http"
	"os"
	"runtime/debug"

	chessgo "github.com/cricklet/chessgo/chess"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

func reader(conn *websocket.Conn) {
	for {
		// read in a message
		messageType, p, err := conn.ReadMessage()
		if err != nil {
			fmt.Println(err)
			return
		}
		// print out that message for clarity
		fmt.Println(string(p))

		if err := conn.WriteMessage(messageType, p); err != nil {
			fmt.Println(err)
			return
		}

	}
}

func wsEndpoint(w http.ResponseWriter, r *http.Request) {
	upgrader.CheckOrigin = func(r *http.Request) bool { return true }

	// upgrade this connection to a WebSocket
	// connection
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Println(err)
	}
	// helpful log statement to show connections
	fmt.Println("Client Connected")

	reader(ws)
}

func serve() {
	fmt.Println("asdf")
	http.Handle("/", http.FileServer(http.Dir("../static")))
	http.ListenAndServe(":8002", nil)
}

func main() {
	defer func() {
		if r := recover(); r != nil {
			chessgo.Log(fmt.Sprint(r))
			chessgo.Log(string(debug.Stack()))
		}
	}()

	args := os.Args[1:]
	if args[0] == "serve" {
		serve()
	} else {
		r := chessgo.Runner{}

		scanner := bufio.NewScanner(os.Stdin)

		done := false
		for !done && scanner.Scan() {
			input := scanner.Text()
			done = r.HandleInputAndReturnDone(input)
		}
	}
}

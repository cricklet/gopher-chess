package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"runtime/debug"
	"strings"

	chessgo "github.com/cricklet/chessgo/chess"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

type UpdateToWeb struct {
	FenBoardString string
	Selection      string
	PossibleMoves  []string
}

func (u UpdateToWeb) String() string {
	return fmt.Sprint("UpdateToWeb: ", u.FenBoardString, ", ", u.Selection, ", ", u.PossibleMoves)
}

type MessageFromWeb struct {
	NewFen    *string
	Selection *string
	Move      *string
}

func (u MessageFromWeb) String() string {
	if u.NewFen != nil {
		return fmt.Sprint("MessageFromWeb newFen: ", *u.NewFen)
	}
	if u.Selection != nil {
		return fmt.Sprint("MessageFromWeb selection: ", *u.Selection)
	}
	if u.Move != nil {
		return fmt.Sprint("MessageFromWeb move: ", *u.Move)
	}
	return "MessageFromWeb unknown"
}
func serve() {
	var upgrader = websocket.Upgrader{}

	var ws = func(w http.ResponseWriter, r *http.Request) {
		runner := chessgo.Runner{}

		c, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			panic(err)
		}

		var sendUpdateToWeb = func(result UpdateToWeb) {
			log.Println("sending", result)
			bytes, err := json.Marshal(result)
			if err != nil {
				panic(err)
			}
			c.WriteMessage(websocket.TextMessage, bytes)
		}

		var handleMessageFromWeb = func(bytes []byte) {
			var message MessageFromWeb
			json.Unmarshal(bytes, &message)
			log.Println("received", message)

			var update UpdateToWeb

			var responses []string

			if message.NewFen != nil {
				for _, command := range []string{
					"isready",
					"uci",
					"ucinewgame",
					fmt.Sprintf("position fen %v", *message.NewFen),
					"go",
					"stop",
				} {
					runner.HandleInput(command)
				}
			} else if message.Selection != nil {
				update.Selection = *message.Selection
				moves := runner.MovesForSelection(*message.Selection)
				update.PossibleMoves = moves
			} else if message.Move != nil {
				runner.PerformMoveFromString(*message.Move)
			}

			for _, r := range responses {
				if strings.HasPrefix(r, "bestmove ") {
					m := strings.TrimPrefix(r, "bestmove ")
					runner.PerformMoveFromString(m)
				}
			}

			update.FenBoardString = runner.FenBoardString()
			sendUpdateToWeb(update)
		}

		defer c.Close()
		for {
			_, message, err := c.ReadMessage()
			if err != nil {
				log.Printf("Error: %v", err)
				break
			} else {
				handleMessageFromWeb(message)
			}
		}
	}

	log.Println("serving")

	router := mux.NewRouter()
	router.HandleFunc("/ws", ws)
	router.PathPrefix("/").Handler(http.FileServer(http.Dir("../static")))
	http.Handle("/", router)
	http.ListenAndServe(":8002", router)
}

func main() {
	defer func() {
		if r := recover(); r != nil {
			log.Println(fmt.Sprint(r))
			log.Println(string(debug.Stack()))
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
			if input == "quit" {
				break
			}
			for _, v := range r.HandleInput(input) {
				fmt.Println(v)
			}
		}
	}
}

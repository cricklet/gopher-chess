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
	FenString     string
	Selection     string
	PossibleMoves []string
	Player        string // white / black
}

func (u UpdateToWeb) String() string {
	return fmt.Sprint("UpdateToWeb: ", u.FenString, ", ", u.Selection, ", ", u.PossibleMoves)
}

type MessageFromWeb struct {
	NewFen    *string
	Selection *string
	Move      *string
	Rewind    *int
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

type LogForwarding struct {
	writeCallback func(message string)
}

func (l *LogForwarding) Println(v ...any) {
	l.writeCallback(fmt.Sprintln(v...))
}
func (l *LogForwarding) Printf(format string, v ...any) {
	l.writeCallback(fmt.Sprintf(format, v...))
}
func (l *LogForwarding) Print(v ...any) {
	l.writeCallback(fmt.Sprint(v...))
}
func serve() {
	var upgrader = websocket.Upgrader{}

	var ws = func(w http.ResponseWriter, r *http.Request) {
		runner := chessgo.Runner{}

		c, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			panic(err)
		}

		runner.Logger = &LogForwarding{
			func(message string) {
				log.Print("logging", message)
				bytes, err := json.Marshal([]string{"server: " + message})
				if err != nil {
					panic(err)
				}
				c.WriteMessage(websocket.TextMessage, bytes)
			},
		}

		var sendUpdateToWeb = func(result UpdateToWeb) {
			runner.Logger.Println("sending", result)
			bytes, err := json.Marshal(result)
			if err != nil {
				panic(err)
			}
			c.WriteMessage(websocket.TextMessage, bytes)
		}

		var handleMessageFromWeb = func(bytes []byte) {
			var message MessageFromWeb
			json.Unmarshal(bytes, &message)
			runner.Logger.Println("received", message)

			var update UpdateToWeb

			if message.NewFen != nil {
				for _, command := range []string{
					"isready",
					"uci",
					"ucinewgame",
					fmt.Sprintf("position fen %v", *message.NewFen),
				} {
					runner.Logger.Println(command)
					runner.HandleInput(command)
				}
			} else if message.Selection != nil {
				if *message.Selection != "" {
					update.Selection = *message.Selection
					update.PossibleMoves = chessgo.MapSlice(
						runner.MovesForSelection(*message.Selection),
						func(v chessgo.FileRank) string {
							return v.String()
						})
				}
			} else if message.Move != nil {
				runner.PerformMoveFromString(*message.Move)

				bestMoveString := chessgo.FindInSlice(runner.HandleInput("go"), func(v string) bool {
					return strings.HasPrefix(v, "bestmove ")
				})
				runner.Logger.Println("found move", bestMoveString)
				if bestMoveString.HasValue() {
					runner.PerformMoveFromString(
						strings.TrimPrefix(bestMoveString.Value(), "bestmove "))
				}
			} else if message.Rewind != nil {
				runner.Rewind(*message.Rewind)
			}

			update.FenString = runner.FenString()
			if runner.Player() == chessgo.WHITE {
				update.Player = "white"
			} else {
				update.Player = "black"
			}
			sendUpdateToWeb(update)
		}

		defer c.Close()
		for {
			_, message, err := c.ReadMessage()
			if err != nil {
				runner.Logger.Printf("Error: %v", err)
				break
			} else {
				handleMessageFromWeb(message)
			}
		}
	}

	var index = func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "../static/index.html")
	}

	log.Println("serving")

	router := mux.NewRouter()
	router.HandleFunc("/ws", ws)
	router.PathPrefix("/static").Handler(
		http.StripPrefix("/static", http.FileServer(http.Dir("../static"))))
	router.PathPrefix("/fen").HandlerFunc(index)
	router.HandleFunc("/", index)
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
		log.Println("starting webserver")
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

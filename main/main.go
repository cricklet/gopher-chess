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
	LastMove      string
	Selection     string
	PossibleMoves []string
	Player        string // white / black
}

func (u UpdateToWeb) String() string {
	return fmt.Sprint("UpdateToWeb: ", u.FenString, ", ", u.LastMove, ", ", u.Selection, ", ", u.PossibleMoves)
}

type MessageFromWeb struct {
	NewFen     *string
	UserPlayer *string
	Selection  *string
	Move       *string
	Rewind     *int
}

func (u MessageFromWeb) String() string {
	if u.NewFen != nil {
		return fmt.Sprint("MessageFromWeb NewFen: ", *u.NewFen)
	}
	if u.UserPlayer != nil {
		return fmt.Sprint("MessageFromWeb UserPlayer: ", *u.UserPlayer)
	}
	if u.Selection != nil {
		return fmt.Sprint("MessageFromWeb Selection: ", *u.Selection)
	}
	if u.Move != nil {
		return fmt.Sprint("MessageFromWeb Move: ", *u.Move)
	}
	if u.Rewind != nil {
		return fmt.Sprint("MessageFromWeb Rewind: ", *u.Rewind)
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
		userPlayer := chessgo.WHITE
		computerPlayer := chessgo.BLACK

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

		var finalizeUpdate = func(update UpdateToWeb) {
			update.FenString = runner.FenString()
			if runner.Player() == chessgo.WHITE {
				update.Player = "white"
			} else {
				update.Player = "black"
			}
			if lastMove := runner.LastMove(); lastMove.HasValue() {
				update.LastMove = lastMove.Value().String()
			}

			runner.Logger.Println("sending", update)
			bytes, err := json.Marshal(update)
			if err != nil {
				panic(err)
			}
			c.WriteMessage(websocket.TextMessage, bytes)
		}

		var performMove = func() {
			bestMoveString := chessgo.FindInSlice(runner.HandleInput("go"), func(v string) bool {
				return strings.HasPrefix(v, "bestmove ")
			})
			runner.Logger.Println("found move", bestMoveString)
			if bestMoveString.HasValue() {
				runner.PerformMoveFromString(
					strings.TrimPrefix(bestMoveString.Value(), "bestmove "))
			}
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
			} else if message.UserPlayer != nil {
				if *message.UserPlayer == "white" {
					userPlayer = chessgo.WHITE
				} else {
					userPlayer = chessgo.BLACK
				}
				computerPlayer = userPlayer.Other()

				if runner.Player() == computerPlayer {
					performMove()
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
				finalizeUpdate(update)

				performMove()
			} else if message.Rewind != nil {
				runner.Rewind(*message.Rewind)
			}

			finalizeUpdate(update)
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
	router.PathPrefix("/white/fen").HandlerFunc(index)
	router.PathPrefix("/black/fen").HandlerFunc(index)
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

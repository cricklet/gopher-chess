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
	. "github.com/cricklet/chessgo/internal/helpers"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

type UpdateToWeb struct {
	FenString     string   `json:"fenString"`
	LastMove      string   `json:"lastMove"`
	Selection     string   `json:"selection"`
	PossibleMoves []string `json:"possibleMoves"`
	Player        string   `json:"player"`
}

func (u UpdateToWeb) String() string {
	return fmt.Sprint("UpdateToWeb: ", u.FenString, ", ", u.LastMove, ", ", u.Selection, ", ", u.PossibleMoves)
}

type MessageFromWeb struct {
	NewFen     *string `json:"newFen"`
	UserPlayer *string `json:"userPlayer"`
	Selection  *string `json:"selection"`
	Move       *string `json:"move"`
	Rewind     *int    `json:"rewind"`
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
		userPlayer := chessgo.White
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
					fmt.Fprintln(os.Stderr, fmt.Sprint("logging: json marshal: ", err))
				}
				err = c.WriteMessage(websocket.TextMessage, bytes)
				if err != nil {
					fmt.Fprintln(os.Stderr, fmt.Sprint("logging: websocket: ", err))
				}
			},
		}

		var finalizeUpdate = func(update UpdateToWeb) {
			update.FenString = runner.FenString()
			if runner.Player() == chessgo.White {
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
				runner.Logger.Println("update: json marshal: ", err)
			}
			err = c.WriteMessage(websocket.TextMessage, bytes)
			if err != nil {
				runner.Logger.Println("websocket: ", err)
			}
		}

		var performMove = func() {
			result, err := runner.HandleInput("go")
			if err != nil {
				runner.Logger.Println("search: ", err)
				return
			}

			bestMoveString := FindInSlice(result, func(v string) bool {
				return strings.HasPrefix(v, "bestmove ")
			})
			runner.Logger.Println("found move", bestMoveString)
			if bestMoveString.HasValue() {
				err := runner.PerformMoveFromString(
					strings.TrimPrefix(bestMoveString.Value(), "bestmove "))
				if err != nil {
					runner.Logger.Println("perform %v: ", bestMoveString.Value(), err)
					return
				}
			}
		}

		var handleMessageFromWeb = func(bytes []byte) {
			var message MessageFromWeb
			err := json.Unmarshal(bytes, &message)
			if err != nil {
				runner.Logger.Println("handleMessageFromWeb: json unmarshal: ", err)
			}
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
					_, err := runner.HandleInput(command)
					if err != nil {
						runner.Logger.Println("setup %v: ", command, err) // TODO reset
					}
				}
			} else if message.UserPlayer != nil {
				if *message.UserPlayer == "white" {
					userPlayer = chessgo.White
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
					result, err := runner.MovesForSelection(*message.Selection)
					if err != nil {
						runner.Logger.Println("moves for %v: ", message.Selection, err)
					}
					update.PossibleMoves = MapSlice(
						result,
						func(v chessgo.FileRank) string {
							return v.String()
						})
				}
			} else if message.Move != nil {
				err := runner.PerformMoveFromString(*message.Move)
				if err != nil {
					runner.Logger.Println("perform %v: ", message.Move, err) // TODO reset
				}
				finalizeUpdate(update)

				performMove()
			} else if message.Rewind != nil {
				err := runner.Rewind(*message.Rewind)
				if err != nil {
					runner.Logger.Println("rewind %v: ", message.Rewind, err) // TODO reset
				}
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
	err := http.ListenAndServe(":8002", router)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

}

func main() {
	defer func() {
		if r := recover(); r != nil {
			fmt.Fprintln(os.Stderr, fmt.Sprint(r))
			fmt.Fprintln(os.Stderr, string(debug.Stack()))
		}
	}()

	args := os.Args[1:]
	if len(args) > 0 && args[0] == "serve" {
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
			result, err := r.HandleInput(input)
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				break
			}
			for _, v := range result {
				fmt.Println(v)
			}
		}
	}
}

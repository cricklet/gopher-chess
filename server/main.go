package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"runtime/debug"
	"strings"
	"time"

	. "github.com/cricklet/chessgo/internal/helpers"
	. "github.com/cricklet/chessgo/internal/runner"
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
	NewFen      *string `json:"newFen"`
	WhitePlayer *string `json:"whitePlayer"`
	BlackPlayer *string `json:"blackPlayer"`
	Selection   *string `json:"selection"`
	Move        *string `json:"move"`
	Ready       *bool   `json:"ready"`
	Rewind      *int    `json:"rewind"`
}

func (u MessageFromWeb) String() string {
	if u.NewFen != nil {
		return fmt.Sprint("MessageFromWeb NewFen: ", *u.NewFen)
	}
	if u.WhitePlayer != nil {
		return fmt.Sprint("MessageFromWeb WhitePlayer: ", *u.WhitePlayer)
	}
	if u.BlackPlayer != nil {
		return fmt.Sprint("MessageFromWeb BlackPlayer: ", *u.BlackPlayer)
	}
	if u.Selection != nil {
		return fmt.Sprint("MessageFromWeb Selection: ", *u.Selection)
	}
	if u.Move != nil {
		return fmt.Sprint("MessageFromWeb Move: ", *u.Move)
	}
	if u.Ready != nil {
		return fmt.Sprint("MessageFromWeb Ready: ", *u.Ready)
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

type PlayerType int

const (
	User PlayerType = iota
	ChessGo
	Stockfish
	Unknown
)

func (t PlayerType) String() string {
	switch t {
	case User:
		return "user"
	case ChessGo:
		return "chessgo"
	case Stockfish:
		return "stockfish"
	case Unknown:
		return "unknown"
	default:
		return "unknown"
	}
}

func PlayerTypeFromString(s string) PlayerType {
	switch s {
	case "user":
		return User
	case "chessgo":
		return ChessGo
	case "stockfish":
		return Stockfish
	}
	return Unknown
}

func main() {
	defer func() {
		if r := recover(); r != nil {
			fmt.Fprintln(os.Stderr, fmt.Sprint(r))
			fmt.Fprintln(os.Stderr, string(debug.Stack()))
		}
	}()

	log.Println("starting webserver")
	var upgrader = websocket.Upgrader{}

	var ws = func(w http.ResponseWriter, r *http.Request) {
		chessGoRunner := ChessGoRunner{}
		stockfishRunner := StockfishRunner{Delay: time.Millisecond * 10}

		playerTypes := [2]PlayerType{User, User}
		ready := false

		var runnerForPlayer = func(p Player) Runner {
			if playerTypes[p] == ChessGo {
				return &chessGoRunner
			} else if playerTypes[p] == Stockfish {
				return &stockfishRunner
			}
			return nil
		}

		c, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			panic(err)
		}

		var log = func(message string) {
			log.Print("logging: ", message)
			bytes, err := json.Marshal([]string{message})
			if err != nil {
				fmt.Fprintln(os.Stderr, fmt.Sprint("logging: json marshal: ", err))
			}
			err = c.WriteMessage(websocket.TextMessage, bytes)
			if err != nil {
				fmt.Fprintln(os.Stderr, fmt.Sprint("logging: websocket: ", err))
			}
		}

		logger := &LogForwarding{
			writeCallback: func(message string) {
				log(fmt.Sprintf("server: %v", message))
			},
		}
		chessGoRunner.Logger = &LogForwarding{
			writeCallback: func(message string) {
				log(fmt.Sprintf("chessgo: %v", message))
			},
		}
		stockfishRunner.Logger = &LogForwarding{
			writeCallback: func(message string) {
				log(fmt.Sprintf("stockfish: %v", message))
			},
		}

		var finalizeUpdate = func(update UpdateToWeb) {
			update.FenString = chessGoRunner.FenString()
			if chessGoRunner.Player() == White {
				update.Player = "white"
			} else {
				update.Player = "black"
			}
			if lastMove := chessGoRunner.LastMove(); lastMove.HasValue() {
				update.LastMove = lastMove.Value().String()
			}

			logger.Println("sending", update)
			bytes, err := json.Marshal(update)
			if err != nil {
				logger.Println("update: json marshal: ", err)
			}
			err = c.WriteMessage(websocket.TextMessage, bytes)
			if err != nil {
				logger.Println("websocket: ", err)
			}
		}

		var performMove = func() bool {
			if !ready {
				return false
			}
			if playerTypes[chessGoRunner.Player()] == User {
				return false
			}

			runner := runnerForPlayer(chessGoRunner.Player())

			results := []string{}

			result, err := runner.HandleInput("position fen " + chessGoRunner.FenStringWithMoves())
			results = append(results, result...)
			if err != nil {
				logger.Println("search: ", err)
				return false
			}

			result, err = runner.HandleInput("go")
			results = append(results, result...)
			if err != nil {
				logger.Println("search: ", err)
				return false
			}

			result, err = runner.HandleInput("stop")
			results = append(results, result...)
			if err != nil {
				logger.Println("search: ", err)
				return false
			}

			bestMoveString := FindInSlice(results, func(v string) bool {
				return strings.HasPrefix(v, "bestmove ")
			})
			if bestMoveString.HasValue() {
				logger.Println("found move", bestMoveString.Value())
				err := chessGoRunner.PerformMoveFromString(
					strings.TrimPrefix(bestMoveString.Value(), "bestmove "))
				if err != nil {
					logger.Println("perform: ", bestMoveString.Value(), err)
					return false
				}
			} else {
				logger.Println("no move found")
				return false
			}

			return true
		}

		var handleMessageFromWeb = func(bytes []byte) {
			var message MessageFromWeb
			err := json.Unmarshal(bytes, &message)
			if err != nil {
				logger.Println("handleMessageFromWeb: json unmarshal: ", err)
			}
			logger.Println("received", message)

			var update UpdateToWeb
			shouldUpdate := false

			if message.NewFen != nil {
				for _, command := range []string{
					"isready",
					"uci",
					"ucinewgame",
					fmt.Sprintf("position fen %v", *message.NewFen),
				} {
					logger.Printf("uci '%v'\n", command)
					_, err := chessGoRunner.HandleInput(command)
					if err != nil {
						logger.Println("setup chessgo: ", command, err) // TODO reset
					}
					_, err = stockfishRunner.HandleInput(command)
					if err != nil {
						logger.Println("setup stockfish: ", command, err) // TODO reset
					}
				}
			} else if message.WhitePlayer != nil {
				playerTypes[White] = PlayerTypeFromString(*message.WhitePlayer)
			} else if message.BlackPlayer != nil {
				playerTypes[Black] = PlayerTypeFromString(*message.BlackPlayer)
			} else if message.Selection != nil {
				if *message.Selection != "" {
					update.Selection = *message.Selection
					result, err := chessGoRunner.MovesForSelection(*message.Selection)
					if err != nil {
						logger.Println("moves for: ", message.Selection, err)
					}
					update.PossibleMoves = result
				}
				shouldUpdate = true
			} else if message.Move != nil {
				err := chessGoRunner.PerformMoveFromString(*message.Move)
				if err != nil {
					logger.Println("perform: ", message.Move, err) // TODO reset
				}
				shouldUpdate = true
			} else if message.Rewind != nil {
				err := chessGoRunner.Rewind(*message.Rewind)
				if err != nil {
					logger.Println("rewind: ", message.Rewind, err) // TODO reset
				}
				shouldUpdate = true
			} else if message.Ready != nil {
				if !ready {
					ready = *message.Ready
					shouldUpdate = true
				}
			}

			if shouldUpdate || (ready && performMove()) {
				finalizeUpdate(update)
			}
		}

		defer c.Close()
		for {
			_, message, err := c.ReadMessage()
			if err != nil {
				logger.Printf("Error: %v", err)
				break
			} else {
				handleMessageFromWeb(message)
			}
		}
	}

	var index = func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "./static/index.html")
	}

	log.Println("serving")

	router := mux.NewRouter()
	router.HandleFunc("/ws", ws)
	router.PathPrefix("/static").Handler(
		http.StripPrefix("/static", http.FileServer(http.Dir("./static"))))
	router.PathPrefix("/{white}/{black}").HandlerFunc(index)
	router.PathPrefix("/{white}/{black}/fen").HandlerFunc(index)
	router.HandleFunc("/", index)
	http.Handle("/", router)
	err := http.ListenAndServe(":8002", router)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

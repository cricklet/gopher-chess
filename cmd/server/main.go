package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"runtime/debug"
	"strconv"

	"github.com/cricklet/chessgo/internal/chessgo"
	. "github.com/cricklet/chessgo/internal/helpers"
	"github.com/cricklet/chessgo/internal/stockfish"
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

	var upgrader = websocket.Upgrader{}

	var ws = func(w http.ResponseWriter, r *http.Request) {
		playerTypes := [2]PlayerType{User, User}
		ready := false

		c, err := upgrader.Upgrade(w, r, nil)
		if !IsNil(err) {
			panic(err)
		}

		var log = func(message string) {
			log.Print("logging: ", message)
			bytes, err := json.Marshal([]string{message})
			if !IsNil(err) {
				fmt.Fprintln(os.Stderr, fmt.Sprint("logging: json marshal: ", err))
			}
			err = c.WriteMessage(websocket.TextMessage, bytes)
			if !IsNil(err) {
				fmt.Fprintln(os.Stderr, fmt.Sprint("logging: websocket: ", err))
			}
		}

		chessGoRunner := chessgo.NewChessGoRunner()

		stockfishRunner, err := stockfish.NewStockfishRunner(stockfish.WithElo(800), stockfish.WithLogger(
			&LogForwarding{
				writeCallback: func(message string) {
					log(fmt.Sprintf("stockfish: %v", message))
				},
			},
		))

		if !IsNil(err) {
			panic(err)
		}

		var runnerForPlayer = func(p Player) Runner {
			if playerTypes[p] == ChessGo {
				return &chessGoRunner
			} else if playerTypes[p] == Stockfish {
				return stockfishRunner
			}
			return nil
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
			if !IsNil(err) {
				logger.Println("update: json marshal: ", err)
			}
			err = c.WriteMessage(websocket.TextMessage, bytes)
			if !IsNil(err) {
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

			err := runner.PerformMoves(chessGoRunner.StartFen, chessGoRunner.MoveHistory())
			if !IsNil(err) {
				logger.Println("setup: ", err)
				return false
			}

			bestMove, _, _, err := runner.Search()
			if !IsNil(err) {
				logger.Println("search: ", err)
				return false
			}

			if bestMove.HasValue() {
				logger.Println("search: ", bestMove.Value())
				err := chessGoRunner.PerformMoveFromString(bestMove.Value())
				if !IsNil(err) {
					logger.Println("perform: ", bestMove.Value(), err)
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
			if !IsNil(err) {
				logger.Println("handleMessageFromWeb: json unmarshal: ", err)
			}
			logger.Println("received", message)

			var update UpdateToWeb
			shouldUpdate := false

			if message.NewFen != nil {
				err := chessGoRunner.SetupPosition(Position{
					Fen:   *message.NewFen,
					Moves: []string{},
				})
				if !IsNil(err) {
					logger.Println("chessgo setup: ", err)
				}
				err = stockfishRunner.SetupPosition(Position{
					Fen:   *message.NewFen,
					Moves: []string{},
				})
				if !IsNil(err) {
					logger.Println("stockfish setup: ", err)
				}
			} else if message.WhitePlayer != nil {
				playerTypes[White] = PlayerTypeFromString(*message.WhitePlayer)
			} else if message.BlackPlayer != nil {
				playerTypes[Black] = PlayerTypeFromString(*message.BlackPlayer)
			} else if message.Selection != nil {
				if *message.Selection != "" {
					update.Selection = *message.Selection
					result, err := chessGoRunner.MovesForSelection(*message.Selection)
					if !IsNil(err) {
						logger.Println("moves for: ", message.Selection, err)
					}
					update.PossibleMoves = result
				}
				shouldUpdate = true
			} else if message.Move != nil {
				err := chessGoRunner.PerformMoveFromString(*message.Move)
				if !IsNil(err) {
					logger.Println("perform: ", message.Move, err) // FUTURE reset
				}
				shouldUpdate = true
			} else if message.Rewind != nil {
				err := chessGoRunner.Rewind(*message.Rewind)
				if !IsNil(err) {
					logger.Println("rewind: ", message.Rewind, err) // FUTURE reset
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
			if !IsNil(err) {
				logger.Printf("Error: %v", err)
				break
			} else {
				handleMessageFromWeb(message)
			}
		}
	}

	var index = func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, RootDir()+"/static/index.html")
	}

	port := 8002

	args := os.Args[1:]
	for _, arg := range args {
		if parsed, err := strconv.ParseInt(arg, 10, 64); err == nil {
			port = int(parsed)
		}
	}

	var err Error
	log.Println("serving at", port)

	router := mux.NewRouter()
	router.HandleFunc("/ws", ws)
	router.PathPrefix("/static").Handler(
		http.StripPrefix("/static", http.FileServer(http.Dir(RootDir()+"/static"))))
	router.PathPrefix("/{white}/{black}").HandlerFunc(index)
	router.PathPrefix("/{white}/{black}/fen").HandlerFunc(index)
	router.HandleFunc("/", index)
	http.Handle("/", router)
	err = Wrap(http.ListenAndServe(fmt.Sprintf(":%v", port), router))
	if !IsNil(err) {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

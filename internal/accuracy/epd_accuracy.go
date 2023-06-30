package accuracy

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/cricklet/chessgo/internal/bitboards"
	"github.com/cricklet/chessgo/internal/game"
	"github.com/cricklet/chessgo/internal/helpers"
	. "github.com/cricklet/chessgo/internal/helpers"
	"github.com/cricklet/chessgo/internal/search"
	"github.com/cricklet/chessgo/internal/stockfish"
)

func EpdToFen(epd string) string {
	parts := strings.Split(epd, " ")
	parts = parts[0:4]
	return strings.Join(parts, " ")
}

func popCapture(moveStr string) (bool, string) {
	if strings.Contains(moveStr, "x") {
		moveStr = strings.Replace(moveStr, "x", "", 1)
		return true, moveStr
	}

	return false, moveStr
}

func popPromotion(moveStr string) (PieceType, string) {
	i := len(moveStr) - 1
	last := moveStr[i:]
	if strings.ToUpper(last) == last {
		pieceType := PieceTypeFromString(last)
		if pieceType != InvalidPiece {
			moveStr := moveStr[0:i]
			pieceType := PieceTypeFromString(last)

			return pieceType, moveStr
		}
	}

	return InvalidPiece, moveStr
}

func popTargetSquare(moveStr string) (FileRank, string, Error) {
	i := len(moveStr) - 2
	target := moveStr[i:]
	prefix := moveStr[0:i]
	fileRank, err := FileRankFromString(target)

	return fileRank, prefix, err
}

func findPiece(pieceStr string, target FileRank, g *game.GameState, b *bitboards.Bitboards) (FileRank, Error) {
	moves := []Move{}
	err := search.GenerateLegalMoves(b, g, &moves)
	if err.HasError() {
		return FileRank{}, err
	}

	matches := []FileRank{}

	disambiguatedRank := Empty[Rank]()
	disambiguatedFile := Empty[File]()

	if len(pieceStr) == 2 {
		char := pieceStr[1]
		if IsRank(char) {
			rank, err := RankFromChar(char)
			if err.HasError() {
				return FileRank{}, err
			}
			disambiguatedRank = Some(rank)
		}
		if IsFile(char) {
			file, err := FileFromChar(char)
			if err.HasError() {
				return FileRank{}, err
			}
			disambiguatedFile = Some(file)
		}
	} else if len(pieceStr) == 1 {
		char := pieceStr[0]
		if IsRank(char) {
			rank, err := RankFromChar(char)
			if err.HasError() {
				return FileRank{}, err
			}
			disambiguatedRank = Some(rank)
		}
	}

	disambiguatedPiece := Empty[PieceType]()
	if len(pieceStr) > 0 {
		char := pieceStr[0:1]
		if strings.ToUpper(char) == char {
			pieceType := PieceTypeFromString(char)
			if pieceType != InvalidPiece {
				disambiguatedPiece = Some(pieceType)
			}
		}
	}

	for _, move := range moves {
		start := FileRankFromIndex(move.StartIndex)
		end := FileRankFromIndex(move.EndIndex)
		startPiece := PieceAtFileRank(g.Board, start)
		if end != target {
			continue
		}
		if disambiguatedPiece.HasValue() && disambiguatedPiece.Value() != startPiece.PieceType() {
			continue
		}
		if disambiguatedPiece.IsEmpty() && startPiece.PieceType() != Pawn {
			continue
		}
		if disambiguatedFile.HasValue() && disambiguatedFile.Value() != start.File {
			continue
		}
		if disambiguatedRank.HasValue() && disambiguatedRank.Value() != start.Rank {
			continue
		}
		matches = append(matches, start)
	}

	if len(matches) == 0 {
		return FileRank{}, Errorf("no piece found")
	}

	if len(matches) > 1 {
		return FileRank{}, Errorf("multiple pieces found for piece %v, target %v, fen %v", pieceStr, target, game.FenStringForBoard(g.Board))
	}

	return matches[0], NilError
}

func popCheck(moveStr string) string {
	return strings.ReplaceAll(moveStr, "+", "")
}

func MoveFromShorthand(moveStr string, g *game.GameState, b *bitboards.Bitboards) (string, Error) {
	moveStr = popCheck(moveStr)
	isCapture, move2 := popCapture(moveStr)
	promotionPieceType, move3 := popPromotion(move2)
	targetFileRank, move4, err := popTargetSquare(move3)
	if err.HasError() {
		return "", err
	}

	startFileRank, err := findPiece(move4, targetFileRank, g, b)
	if err.HasError() {
		return "", err
	}

	move := g.MoveFromString(startFileRank.String() +
		targetFileRank.String() +
		promotionPieceType.String())

	if isCapture && !move.MoveType.Captures() {
		return "", Errorf("move should be a capture")
	}

	return move.String(), NilError
}

func MovesFromEpd(prefix string, epd string, g *game.GameState, b *bitboards.Bitboards) ([]string, Error) {
	if !strings.Contains(epd, prefix+" ") {
		return []string{}, NilError
	}
	end := strings.Split(epd, prefix+" ")[1]
	movesStr := strings.Split(end, ";")[0]

	moves := []string{}

	for _, moveStr := range strings.Split(movesStr, ", ") {
		move, err := MoveFromShorthand(moveStr, g, b)
		if err.HasError() {
			return []string{}, err
		}

		moves = append(moves, move)
	}

	return moves, NilError
}

type EpdCacheResult int

const (
	EpdCacheResultSuccess EpdCacheResult = iota
	EpdCacheResultFailure
	EpdCacheResultAmbiguous
)

type EpdResult struct {
	Epd string `json:"epd"`

	BestMoves  []string `json:"best_moves"`
	AvoidMoves []string `json:"avoid_moves"`

	StockfishScores map[string]int `json:"stockfish_scores"`
	StockfishMove   string         `json:"stockfish_move"`
	StockfishResult EpdCacheResult `json:"stockfish_result"`
	StockfishDepth  int            `json:"stockfish_depth"`
}

func calculateSuccess(move string, bestMoves []string, avoidMoves []string) bool {
	if len(bestMoves) > 0 && !Contains(bestMoves, move) {
		return false
	}
	if len(avoidMoves) > 0 && Contains(avoidMoves, move) {
		return false
	}
	return true
}

func CalculateDepthForEpdSuccess(
	logger *LiveLogger,
	stock *stockfish.StockfishRunner,
	epd string,
	bestMoves []string,
	avoidMoves []string,
	maxDepth Optional[int],
) (string, int, Error) {
	depth := 0
	bestMove := ""

	consecutiveSuccesses := map[int]bool{}

	if stock.MultiPVEnabled {
		return "", 0, Errorf("MultiPV must be disabled")
	}

	requireSuccesses := 6

	err := stock.SearchUnlimitedRaw(
		func(line string) (LoopResult, Error) {
			move, _, err := stockfish.MoveAndScoreFromInfoLine(line)
			if err.HasError() {
				return LoopBreak, err
			}

			if !move.HasValue() {
				return LoopContinue, NilError
			}
			depth, err = stockfish.DepthFromInfoLine(line)
			if err.HasError() {
				return LoopBreak, err
			}

			if maxDepth.HasValue() && depth >= maxDepth.Value() {
				return LoopBreak, NilError
			}

			if calculateSuccess(move.Value(), bestMoves, avoidMoves) {
				consecutiveSuccesses[depth] = true
			} else {
				consecutiveSuccesses = map[int]bool{}
			}

			if len(consecutiveSuccesses) >= requireSuccesses {
				bestMove = move.Value()
				return LoopBreak, NilError
			}

			return LoopContinue, NilError
		},
	)

	logger.Println(consecutiveSuccesses)

	if err.HasError() {
		return "", depth, err
	}

	return bestMove, depth - requireSuccesses + 1, NilError
}

func CalculateScoreForEveryMove(
	logger *LiveLogger,
	stock *stockfish.StockfishRunner,
	goalDepth int,
	moveToPrioritize string,
	fen string,
	g *game.GameState,
	b *bitboards.Bitboards,
) (map[string]int, Error) {
	scores := map[string]int{}

	if stock.MultiPVEnabled {
		return scores, Errorf("MultiPV must be disabled")
	}

	moves := []Move{}
	err := search.GenerateLegalMoves(b, g, &moves)
	if err.HasError() {
		return scores, err
	}

	prioritized := helpers.MoveToFront(&moves, func(move Move) bool {
		return move.String() == moveToPrioritize
	})

	if !prioritized {
		return scores, Errorf("move to prioritize not found")
	}

	for i, move := range moves {
		err := stock.SetupPosition(Position{
			Fen: fen,
			Moves: []string{
				move.String(),
			},
		})
		if err.HasError() {
			return scores, err
		}

		_, enemyScore, err := stock.SearchDepth(goalDepth)
		if err.HasError() {
			return scores, err
		}

		if enemyScore.IsEmpty() {
			return scores, Errorf("no score found for %v", move.String())
		}

		score := -enemyScore.Value()

		logger.Printf("(%v / %v) score for %v is %v\n", i+1, len(moves), move.String(), search.ScoreString(score))
		scores[move.String()] = score
	}

	return scores, NilError
}

type Epd struct {
	epd string
	fen string

	bestMoves  []string
	avoidMoves []string

	game      *game.GameState
	bitboards *bitboards.Bitboards
}

func ParseEpd(epd string) (*Epd, Error) {
	fen := EpdToFen(epd)
	game, err := game.GamestateFromFenString(fen)
	if err.HasError() {
		panic(err)
	}

	bitboards := game.CreateBitboards()

	bestMoves, err := MovesFromEpd("bm", epd, game, bitboards)
	if err.HasError() {
		return nil, err
	}

	avoidMoves, err := MovesFromEpd("am", epd, game, bitboards)
	if err.HasError() {
		return nil, err
	}

	if len(bestMoves) == 0 && len(avoidMoves) == 0 {
		return nil, Errorf("no bm or am in epd: %v", epd)
	}

	return &Epd{
		epd:        epd,
		fen:        fen,
		bestMoves:  bestMoves,
		avoidMoves: avoidMoves,
		game:       game,
		bitboards:  bitboards,
	}, NilError
}

func CalculateEpdResult(stock *stockfish.StockfishRunner, logger *LiveLogger, epd string) EpdResult {
	parsed, err := ParseEpd(epd)
	if err.HasError() {
		panic(err)
	}

	err = stock.SetupPosition(Position{Fen: parsed.fen})
	if err.HasError() {
		panic(err)
	}

	// err = stock.SetHashSize(1024 * 5)
	// if err.HasError() {
	// 	panic(err)
	// }

	result := EpdResult{}
	result.Epd = epd
	result.BestMoves = parsed.bestMoves
	result.AvoidMoves = parsed.avoidMoves

	// defer profile.Start(profile.ProfilePath(RootDir() + "/data/EpdCacheProfile")).Stop()

	move, depth, err := CalculateDepthForEpdSuccess(
		logger,
		stock,
		epd,
		parsed.bestMoves,
		parsed.avoidMoves,
		Some(28),
	)

	if !calculateSuccess(move, parsed.bestMoves, parsed.avoidMoves) {
		result.StockfishMove = move
		result.StockfishScores = nil
		result.StockfishResult = EpdCacheResultFailure
		result.StockfishDepth = depth
		return result
	}

	logger.SetFooter("", 0)
	logger.Println(fmt.Sprintf("found correct move w/ depth %v", depth))

	moveToScore, err := CalculateScoreForEveryMove(
		logger,
		stock,
		depth+3,
		move,
		parsed.fen,
		parsed.game,
		parsed.bitboards,
	)
	if err.HasError() {
		panic(err)
	}

	result.StockfishMove = move
	result.StockfishScores = moveToScore
	result.StockfishDepth = depth

	bestMove := MaxInMap(moveToScore)
	bestMoveIsCertain := calculateSuccess(bestMove, parsed.bestMoves, parsed.avoidMoves)

	if bestMoveIsCertain {
		result.StockfishResult = EpdCacheResultSuccess
	} else {
		result.StockfishResult = EpdCacheResultAmbiguous
	}

	return result
}

func LoadEpd(path string) ([]string, Error) {
	file, err := WrapReturn(os.Open(path))
	if err.HasError() {
		return []string{}, err
	}

	results := []string{}

	fscanner := bufio.NewScanner(file)
	for fscanner.Scan() {
		line := fscanner.Text()
		line = strings.TrimSpace(line)
		if len(line) == 0 {
			continue
		}

		results = append(results, line)
	}

	return results, NilError
}

func SearchEpd(runner Runner, epd string) (string, bool, int, Error) {
	parsed, err := ParseEpd(epd)
	if err.HasError() {
		return "", false, 0, err
	}

	err = runner.SetupPosition(Position{Fen: parsed.fen})
	if err.HasError() {
		return "", false, 0, err
	}

	move, _, depth, err := runner.Search()

	if move.IsEmpty() {
		return "", false, depth, Errorf("no moves found")
	}

	success := calculateSuccess(move.Value(), parsed.bestMoves, parsed.avoidMoves)
	return move.Value(), success, depth, NilError
}

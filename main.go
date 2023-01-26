package main

import (
	"bufio"
	"fmt"
	"os"
	"runtime/debug"
	"strings"
)

type File int
type Rank int

func (f File) str() string {
	return [8]string{
		"a", "b", "c", "d", "e", "f", "g", "h",
	}[f]
}
func (r Rank) str() string {
	return [8]string{
		"1", "2", "3", "4", "5", "6", "7", "8",
	}[r]
}

type Piece int

const (
	XX Piece = iota
	WR
	WN
	WB
	WK
	WQ
	WP
	BR
	BN
	BB
	BK
	BQ
	BP
)

func (p Piece) str() string {
	return []string{
		" ",
		"R",
		"N",
		"B",
		"K",
		"Q",
		"P",
		"r",
		"n",
		"b",
		"k",
		"q",
		"p",
	}[p]
}

func (p Piece) isWhite() bool {
	return p <= WP && p >= WR
}

func (p Piece) isBlack() bool {
	return p <= BP && p >= BR
}

func (p Piece) isEmpty() bool {
	return p == XX
}

type BoardArray [64]Piece

func (b BoardArray) str() string {
	result := ""
	for r := 0; r < 8; r++ {
		row := b[r*8 : (r+1)*8]
		for _, p := range row {
			result += p.str()
		}
		result += "\n"
	}
	return result
}

type Game struct {
}

func main() {
	defer func() {
		if r := recover(); r != nil {
			fmt.Println(string(debug.Stack()))
		}
	}()

	var testFile File = 0
	var testRank Rank = 1

	scanner := bufio.NewScanner(os.Stdin)
	done := false
	for !done && scanner.Scan() {
		input := scanner.Text()
		if input == "uci" {
			fmt.Println("name chess-go")
			fmt.Println("id author Kenrick Rilee")
			fmt.Println("uciok")
		} else if input == "isready" {
			fmt.Println("readyok")
		} else if strings.HasPrefix(input, "go") {
			fmt.Printf("bestmove %s%s%s%s\n", testFile.str(), testRank.str(), testFile.str(), (testRank + 1).str())
			testFile++
		} else if input == "quit" {
			done = true
		}
	}
}

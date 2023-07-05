package helpers

import (
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"

	"github.com/davecgh/go-spew/spew"
)

type LoopResult int

const (
	LoopContinue LoopResult = iota
	LoopBreak
)

type Success bool

func Ignore(t ...any) {
}

func AsyncSend[T any](c *chan T, t T) {
	select {
	case *c <- t:
		{
		}
	default:
		{
		}
	}
}

func Ellipses(s string, maxLength int) string {
	if len(s) > maxLength {
		for i := maxLength - 1; i >= 0; i-- {
			if s[i] == ' ' {
				return s[:i] + "..."
			}
		}

		// return s[:maxLength-3] + "..."
	}
	return s
}

func ValuesAllEqual[K comparable, T comparable](m map[K]T) bool {
	var initial T
	for _, v := range m {
		initial = v
		break
	}

	for _, v := range m {
		if v != initial {
			return false
		}
	}

	return true
}

func GetWithDefault[K comparable, T any](m map[K]T, key K, d T) T {
	if v, ok := m[key]; ok {
		return v
	}
	return d
}

func GetHostName() (string, Error) {
	hostname, err := os.Hostname()
	if err != nil {
		return "", Wrap(err)
	}
	return hostname, NilError
}

func Indent(s string, x string) string {
	return strings.ReplaceAll(s, "\n", "\n"+x)
}

func IndentMany(indent string, xs ...any) string {
	return strings.ReplaceAll(strings.Join(
		MapSlice(xs, func(x any) string {
			return fmt.Sprint(x)
		}), " "), "\n", "\n"+indent)
}

func Last[T any](ts []T) T {
	return ts[len(ts)-1]
}

func LastN[T any](ts []T, n int) []T {
	return ts[MaxInt(len(ts)-n, 0):]
}

func ParseInt(s string) (int, Error) {
	v, err := strconv.Atoi(s)
	if err != nil {
		return 0, Wrap(err)
	}
	return v, NilError
}

func Clone[T any](ts []T) []T {
	return append([]T(nil), ts...)
}

func MapSlice[T, U any](ts []T, f func(T) U) []U {
	us := make([]U, len(ts))
	for i := range ts {
		us[i] = f(ts[i])
	}
	return us
}

func ConcatStringify[T fmt.Stringer](ts []T) string {
	return strings.Join(
		MapSlice(ts, func(t T) string { return t.String() }),
		", ")
}

func FilterSlice[T any](ts []T, f func(T) bool) []T {
	filtered := []T{}
	for i := range ts {
		if f(ts[i]) {
			filtered = append(filtered, ts[i])
		}
	}
	return filtered
}
func FindInSlice[T any](ts []T, f func(T) bool) Optional[T] {
	for i := range ts {
		if f(ts[i]) {
			return Some(ts[i])
		}
	}
	return Empty[T]()
}
func PopValue[T any](ts []T, d T) (T, []T) {
	if len(ts) == 0 {
		return d, ts
	}
	return ts[len(ts)-1], ts[:len(ts)-1]
}
func PopOptional[T any](ts []T) (Optional[T], []T) {
	if len(ts) == 0 {
		return Empty[T](), ts
	}
	return Some(ts[len(ts)-1]), ts[:len(ts)-1]
}
func PopPtr[T any](ts []*T) (*T, []*T) {
	if len(ts) == 0 {
		return nil, ts
	}
	return ts[len(ts)-1], ts[:len(ts)-1]
}

func IndexOf[T any](ts []T, f func(T) bool) Optional[int] {
	for i := range ts {
		if f(ts[i]) {
			return Some(i)
		}
	}
	return Empty[int]()
}

func Contains[T comparable](ts []T, t T) bool {
	return FindInSlice(ts, func(v T) bool {
		return v == t
	}).HasValue()
}

func ReduceSlice[T, U any](ts []T, initial U, f func(U, T) U) U {
	u := initial
	for _, t := range ts {
		u = f(u, t)
	}
	return u
}

func MoveToFront[T any](ts *[]T, f func(T) bool) bool {
	for i, t := range *ts {
		if f(t) {
			*ts = append((*ts)[:i], (*ts)[i+1:]...)
			*ts = append([]T{t}, *ts...)
			return true
		}
	}
	return false
}

func SortMaxFirst[T any](ts *[]T, f func(T) int) {
	sort.SliceStable(*ts, func(i, j int) bool {
		return f((*ts)[j]) < f((*ts)[i])
	})
}
func SortMinFirst[T any](ts *[]T, f func(T) int) {
	sort.SliceStable(*ts, func(i, j int) bool {
		return f((*ts)[i]) < f((*ts)[j])
	})
}
func IndexOfMax[T any](ts []T, f func(T) int) int {
	bestValue := f(ts[0])
	bestIndex := 0

	for i, t := range ts {
		newValue := f(t)
		if newValue > bestValue {
			bestValue = newValue
			bestIndex = i
		}
	}
	return bestIndex
}

func MaxInMap(m map[string]int) string {
	max := Empty[int]()
	maxKey := ""
	for k, v := range m {
		if !max.HasValue() || v > max.Value() {
			max = Some(v)
			maxKey = k
		}
	}
	return maxKey
}

func RandomInt(min int, max int) int {
	return min + rand.Intn(max-min+1)
}

func PickRandom[T any](ts []T) T {
	return ts[RandomInt(0, len(ts)-1)]
}

func IndexOfMin[T any](ts []T, f func(T) int) int {
	bestValue := f(ts[0])
	bestIndex := 0

	for i, t := range ts {
		newValue := f(t)
		if newValue < bestValue {
			bestValue = newValue
			bestIndex = i
		}
	}
	return bestIndex
}

func SingleUint8(indexFromTheRight int) uint8 {
	return 1 << indexFromTheRight
}

var ReverseBitsCache = func() [256]uint8 {
	result := [256]uint8{}
	for i := uint8(0); ; i++ {
		reversed := uint8(0)
		for bit := 0; bit < 8; bit++ {
			if i&SingleUint8(bit) > 0 {
				reversed |= SingleUint8(7 - bit)
			}
		}
		result[i] = reversed

		if i == uint8(255) {
			break
		}
	}
	return result
}()

func ReverseBits(n uint8) uint8 {
	return ReverseBitsCache[n]
}

type Optional[T any] struct {
	_hasValue bool
	_t        T
}

func (o Optional[T]) String() string {
	if o._hasValue {
		return fmt.Sprintf("Some(%v)", o._t)
	}
	return "None"
}

func Some[T any](t T) Optional[T] {
	return Optional[T]{true, t}
}

func Empty[T any]() Optional[T] {
	return Optional[T]{}
}

func (o Optional[T]) IsEmpty() bool {
	return !o._hasValue
}

func (o Optional[T]) HasValue() bool {
	return !o.IsEmpty()
}

func (o Optional[T]) Value() T {
	return o._t
}

func (o Optional[T]) ValueOr(t T) T {
	if o._hasValue {
		return o._t
	}
	return t
}

func MapOptional[T, V any](o Optional[T], f func(T) V) Optional[V] {
	if o.HasValue() {
		return Some(f(o.Value()))
	}
	return Empty[V]()
}

func AbsDiff(x int, y int) int {
	if x < y {
		return y - x
	}
	return x - y
}

func MinInt(x int, y int) int {
	if x < y {
		return x
	}
	return y
}

func MaxInt(x int, y int) int {
	if x > y {
		return x
	}
	return y
}

func FlipArray(array [8][8]int) [8][8]int {
	result := [8][8]int{}
	for i := 0; i < 8; i++ {
		for j := 0; j < 8; j++ {
			result[i][j] = array[7-i][j]
		}
	}
	return result
}

var (
	_, b, _, _ = runtime.Caller(0)
	_basepath  = filepath.Join(filepath.Dir(b), "../..")
)

func RootDir() string {
	return _basepath
}

const hintColor = "\033[38;5;240m"
const resetColors = "\033[0m"

func HintText(text string) string {
	return hintColor + text + resetColors
}

func PrettyPrint(t any) string {
	return spew.Sdump(t)
}

func bToKb(b uint64) uint64 {
	return b / 1024
}

func MemUsageString() string {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	// For info on each, see: https://golang.org/pkg/runtime/#MemStats
	v := fmt.Sprintf("%vkb, total %vkb", bToKb(m.Alloc), bToKb(m.TotalAlloc))

	return v
}

func PrintColumns(values []string, sizes []int, separator string) string {
	result := ""
	for i, v := range values {
		if i < len(sizes) {
			result += fmt.Sprintf("%-"+strconv.Itoa(sizes[i])+"s", v)
		} else {
			result += v
		}
		if i < len(values)-1 {
			result += separator
		}
	}
	return result
}

type NoCopy struct{}

func (*NoCopy) Lock()   {}
func (*NoCopy) Unlock() {}

type Either[A, B any] struct {
	Left  Optional[A]
	Right Optional[B]
}

func (e Either[A, B]) String() string {
	if e.Left.HasValue() {
		return fmt.Sprintf("Left(%v)", e.Left.Value())
	} else if e.Right.HasValue() {
		return fmt.Sprintf("Right(%v)", e.Right.Value())
	} else {
		return "Empty"
	}
}

func Left[A, B any](a A) Either[A, B] {
	return Either[A, B]{Some(a), Empty[B]()}
}

func Right[A, B any](b B) Either[A, B] {
	return Either[A, B]{Empty[A](), Some(b)}
}

func EmptyEither[A, B any]() Either[A, B] {
	return Either[A, B]{Empty[A](), Empty[B]()}
}

func (e Either[A, B]) HasLeft() bool {
	return e.Left.HasValue()
}

func (e Either[A, B]) HasRight() bool {
	return e.Right.HasValue()
}

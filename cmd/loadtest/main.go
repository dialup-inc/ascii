package main

import (
	"context"
	"flag"
	"fmt"
	"image/color"
	"math"
	"math/rand"
	"net"
	"os"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/dialup-inc/ascii/term"
	"github.com/gorilla/websocket"
)

var (
	Green = color.RGBA{0x00, 0xFF, 0x00, 0xFF}
	Blue  = color.RGBA{0x00, 0x00, 0xFF, 0xFF}
)

func main() {
	var (
		wsURL       = flag.String("ws", "wss://roulette.dialup.com/ws", "signaling server websocket url")
		concurrency = flag.Int64("p", 10, "number of simultaneous tests")
	)
	flag.Parse()

	ctx := context.Background()

	var resultsMu sync.Mutex
	var results []Result

	var testsActive int64

	run := func(ctx context.Context) {
		ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
		defer cancel()

		// Add some jitter
		delay := rand.Int63n(int64(5 * time.Second))
		time.Sleep(time.Duration(delay))

		res := RunTest(ctx, *wsURL)

		atomic.AddInt64(&testsActive, -1)

		resultsMu.Lock()
		results = append(results, res)
		resultsMu.Unlock()
	}

	ansiOut := term.ANSI{os.Stdout}
	title := func(s string, c color.Color) {
		ansiOut.Bold()
		ansiOut.Foreground(c)
		fmt.Println(s)
		ansiOut.ForegroundReset()
		ansiOut.Normal()
	}
	summary := func() {
		ansiOut.Clear()
		ansiOut.CursorPosition(1, 1)

		active := atomic.LoadInt64(&testsActive)

		var durations []float64
		var errs []error

		resultsMu.Lock()
		for _, r := range results {
			durations = append(durations, float64(r.Duration))
			errs = append(errs, r.Err)
		}
		resultsMu.Unlock()

		title("Running Load Test...", Green)
		fmt.Println("")

		title("Jobs:", Blue)
		fmt.Println("Active    = ", active)
		fmt.Println("Target    = ", *concurrency)
		fmt.Println("Completed = ", len(durations))
		fmt.Println("")

		if len(durations) == 0 {
			return
		}

		// Duration statistics
		durMed := time.Duration(percentile(durations, 0.5))
		dur95 := time.Duration(percentile(durations, 0.95))

		title("Duration:", Blue)
		fmt.Println("median = ", durMed)
		fmt.Println("95%    = ", dur95)
		fmt.Println("")

		var errRate float64
		for _, e := range errs {
			if e != nil {
				errRate++
			}
		}
		errRate /= float64(len(errs))

		title("Errors:", Blue)
		fmt.Printf("rate = %.02f%%\n", errRate*100)
		fmt.Println("")

		top := topErrs(errs)
		for _, e := range top {
			if e.Err == nil {
				continue
			}
			fmt.Printf("%d  | %v\n", e.Count, e.Err)
		}
	}

	for range time.Tick(1 * time.Second) {
		active := atomic.LoadInt64(&testsActive)
		for i := active; i < *concurrency; i++ {
			atomic.AddInt64(&testsActive, 1)
			go run(ctx)
		}

		summary()
	}
}

type errCountSlice []errCount

func (s errCountSlice) Len() int           { return len(s) }
func (s errCountSlice) Less(i, j int) bool { return s[i].Count < s[j].Count }
func (s errCountSlice) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }

type errCount struct {
	Err   error
	Count int
}

func topErrs(errs []error) []errCount {
	counts := map[string]errCount{}

	for _, e := range errs {
		msg := "<nil>"
		if ne, ok := e.(net.Error); ok && ne.Timeout() {
			msg = "i/o timeout" // remove ip because it's spammy otherwise
		} else if e != nil {
			msg = e.Error()
		}

		c, exist := counts[msg]
		if !exist {
			c.Err = e
		}

		c.Count++
		counts[msg] = c
	}

	var slice errCountSlice
	for _, c := range counts {
		slice = append(slice, c)
	}

	sort.Sort(slice)

	return slice
}

func percentile(xs []float64, perc float64) float64 {
	if len(xs) == 0 {
		return math.NaN()
	}

	size := float64(len(xs))
	sort.Float64s(xs)

	i := perc * size
	if i < 1.0 {
		return xs[0]
	} else if i >= size {
		return xs[len(xs)-1]
	} else {
		frac := i - math.Floor(i)
		a := xs[int(i)-1]
		b := xs[int(i)]
		return a + frac*(b-a)
	}
}

func RunTest(ctx context.Context, wsURL string) Result {
	start := time.Now()
	err := fakeMatch(ctx, wsURL)
	duration := time.Since(start)

	return Result{
		Err:      err,
		Duration: duration,
	}
}

// fakeMatch follows the same protocol as ascii.Match but doesn't spin up a PeerConnection
func fakeMatch(ctx context.Context, wsURL string) error {
	ws, _, err := websocket.DefaultDialer.DialContext(ctx, wsURL, nil)
	if err != nil {
		return err
	}
	defer func() {
		deadline := time.Now().Add(100 * time.Millisecond)
		msg := websocket.FormatCloseMessage(websocket.CloseNormalClosure, "")
		ws.WriteControl(websocket.CloseMessage, msg, deadline)

		ws.Close()
	}()

	msg := &signalMsg{}
	for {
		if err := ctx.Err(); err != nil {
			return err
		}

		if err := ws.ReadJSON(msg); err != nil {
			return err
		}

		switch msg.Type {
		case "requestOffer":
			if err := ws.WriteJSON(signalMsg{
				Type:    "offer",
				Payload: "offer",
			}); err != nil {
				return err
			}

		case "offer":
			if err := ws.WriteJSON(signalMsg{
				Type:    "answer",
				Payload: "answer",
			}); err != nil {
				return err
			}

			return nil
		case "answer":
			if err := ws.WriteJSON(signalMsg{
				Type:    "answerAck",
				Payload: "answerAck",
			}); err != nil {
				return err
			}

			return nil
		default:
			return fmt.Errorf("unknown signaling message %v", msg.Type)
		}

	}
}

type signalMsg struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
}

type Result struct {
	Duration time.Duration
	Err      error
}

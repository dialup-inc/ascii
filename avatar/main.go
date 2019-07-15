package main

import (
	"encoding/json"
	"flag"
	"image"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/dialup-inc/ascii/term"
	"github.com/dialup-inc/ascii/ui"

	"image/color"
	_ "image/jpeg"
	_ "image/png"
)

type profile struct {
	AvatarURL string `json:"avatar_url"`
}

func fetchAvatar(username string) (image.Image, error) {
	resp, err := http.Get("https://api.github.com/users/" + username)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var prof profile
	if err := json.NewDecoder(resp.Body).Decode(&prof); err != nil {
		return nil, err
	}

	resp, err = http.Get(prof.AvatarURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	img, _, err := image.Decode(resp.Body)
	if err != nil {
		return nil, err
	}

	return img, nil
}

func main() {
	flag.Parse()

	username := flag.Arg(0)

	img, err := fetchAvatar(username)
	if err != nil {
		log.Fatal(err)
	}

	ansi := term.ANSI{os.Stdout}

	defer func() {
		ansi.ShowCursor()
		ansi.Reset()
		os.Stdout.Sync()
	}()

	ws, err := term.GetWinSize()
	if err != nil {
		log.Fatal(err)
	}
	aspect := float64(ws.Height) * float64(ws.Cols) / float64(ws.Rows) / float64(ws.Width)

	ansi.ResizeWindow(10, int(10*aspect))

	// Let the resize happen
	time.Sleep(500 * time.Millisecond)

	ws, err = term.GetWinSize()
	if err != nil {
		log.Fatal(err)
	}

	ansi.Background(color.RGBA{0, 0, 0, 255})
	ansi.CursorPosition(1, 1)

	imgANSI := ui.Image2ANSI(img, ws.Cols, ws.Rows, aspect, false)
	os.Stdout.Write(imgANSI)

	ansi.HideCursor()

	buf := make([]byte, 1)
	os.Stdin.Read(buf)
}

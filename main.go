package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"image"
	"image/color"
	"log"
	"net"
	"net/http"
	"os/exec"
	"runtime"
	"sync"
	"time"

	"github.com/buger/goterm"
	"github.com/qeesung/image2ascii/convert"

	"github.com/pions/webrtc"
	"github.com/pions/webrtc/pkg/ice"
	"github.com/pions/webrtc/pkg/rtcp"
)

type Demo struct {
	Colored   bool
	RTCConfig webrtc.RTCConfiguration

	imgMu sync.Mutex
	img   *image.RGBA

	width  int
	height int

	connMu sync.Mutex
	conn   *webrtc.RTCPeerConnection
}

func (d *Demo) HandleStart(w http.ResponseWriter, r *http.Request) {
	var offer webrtc.RTCSessionDescription
	if err := json.NewDecoder(r.Body).Decode(&offer); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	answer, err := d.startConn(offer)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	json.NewEncoder(w).Encode(answer)
}

func (d *Demo) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/start" {
		d.HandleStart(w, r)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(html))
}

func (d *Demo) newConn() (*webrtc.RTCPeerConnection, error) {
	d.connMu.Lock()
	defer d.connMu.Unlock()

	if d.conn != nil {
		return nil, errors.New("another peer connection is connected")
	}

	conn, err := webrtc.New(d.RTCConfig)
	if err != nil {
		return nil, err
	}
	d.conn = conn

	return conn, nil
}

func (d *Demo) startConn(offer webrtc.RTCSessionDescription) (answer webrtc.RTCSessionDescription, err error) {
	ctx, cancel := context.WithCancel(context.Background())

	conn, err := d.newConn()
	if err != nil {
		return answer, err
	}

	conn.OnICEConnectionStateChange = func(s ice.ConnectionState) {
		if s == ice.ConnectionStateClosed || s == ice.ConnectionStateFailed {
			d.connMu.Lock()
			if conn == d.conn {
				d.conn = nil
			}
			d.connMu.Unlock()

			cancel()
		}
	}

	var once sync.Once
	conn.OnTrack = func(track *webrtc.RTCTrack) {
		once.Do(func() {
			d.handleTrack(ctx, track)
		})
	}

	if err := d.conn.SetRemoteDescription(offer); err != nil {
		return answer, err
	}

	answer, err = d.conn.CreateAnswer(nil)
	if err != nil {
		return answer, err
	}

	return answer, err
}

func (d *Demo) handleTrack(ctx context.Context, track *webrtc.RTCTrack) {
	pipeline := CreatePipeline(track.Codec.Name, d.width, d.height)
	pipeline.Start()

	// Send PLIs every once in a while
	go func() {
		ticker := time.NewTicker(time.Second * 3)
		for {
			select {
			case <-ctx.Done():
				return

			case <-ticker.C:
				pli := &rtcp.PictureLossIndication{MediaSSRC: track.Ssrc}
				if err := d.conn.SendRTCP(pli); err != nil {
					fmt.Println(err)
				}
			}
		}
	}()

	// Read raw video frames from the pipeline
	go func() {
		stride := d.width * 4
		frame := make([]byte, stride*d.height)
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}

			_, err := pipeline.Pull(frame)
			if err != nil {
				fmt.Println(err)
			}

			d.imgMu.Lock()
			for i := 0; i < len(frame); i += 4 {
				x := (i % stride) / 4
				y := i / stride

				r := frame[i+0]
				g := frame[i+1]
				b := frame[i+2]
				a := frame[i+3]
				d.img.SetRGBA(x, y, color.RGBA{r, g, b, a})
			}
			d.imgMu.Unlock()
		}
	}()

	// Send encoded packets to the pipeline
	for p := range track.Packets {
		pipeline.Push(p.Raw)
	}
}

// runs in own goroutine
func (d *Demo) printLoop() {
	opts := convert.DefaultOptions
	conv := convert.NewImageConverter()

	goterm.Clear()

	for range time.Tick(time.Second / 5) {
		d.imgMu.Lock()
		ascii := conv.Image2ASCIIString(d.img, &opts)
		d.imgMu.Unlock()

		opts.FixedWidth = goterm.Width()
		opts.FixedHeight = goterm.Height() - 1
		opts.Colored = d.Colored

		goterm.MoveCursor(1, 1)
		goterm.Print(ascii)
		goterm.Flush()
	}
}

func NewDemo(width, height int) *Demo {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	d := &Demo{
		img:    img,
		width:  width,
		height: height,
	}
	go d.printLoop()
	return d
}

func main() {
	var (
		color = flag.Bool("color", true, "whether to render image with colors")
	)
	flag.Parse()

	webrtc.RegisterDefaultCodecs()

	demo := NewDemo(100, 75)
	demo.RTCConfig = webrtc.RTCConfiguration{
		IceServers: []webrtc.RTCIceServer{
			{URLs: []string{"stun:stun.l.google.com:19302"}},
		},
	}
	demo.Colored = *color

	// Start server on an open port
	l, err := net.Listen("tcp", ":0")
	if err != nil {
		log.Fatal(err)
	}
	port := l.Addr().(*net.TCPAddr).Port
	siteURL := fmt.Sprintf("http://localhost:%d/", port)
	open(siteURL)

	if err := http.Serve(l, demo); err != nil {
		log.Fatal(err)
	}
}

func open(url string) {
	var err error

	switch runtime.GOOS {
	case "linux":
		err = exec.Command("xdg-open", url).Start()
	case "windows":
		err = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		err = exec.Command("open", url).Start()
	default:
		err = fmt.Errorf("unsupported platform")
	}
	if err != nil {
		log.Fatal(err)
	}
}

const html = `<!doctype html>
<html>
<head>
	<title>ASCII-RTC</title>
	<style>
		body {
			font-family: sans-serif;
		}
		#create {
			font-size: 24px;
			border-radius: 10px;
			background: white;
			border: solid #CCC 1px;
			margin: 15px;
			padding: 15px 40px;
		}
		#video {
			width: 100%;
			display: none;
		}
	</style>
</head>
<body>
	<div id="main">
		<video id="video" autoplay muted></video> <br />
		<button id="create" onclick="window.createSession()">Start</button>
	</div>

	<div id="signalingContainer" style="display: none">
	Browser base64 Session Description <textarea id="localSessionDescription" readonly="true"></textarea> <br />
	Golang base64 Session Description: <textarea id="remoteSessionDescription"></textarea> <br/>
	<button onclick="window.startSession()"> Start Session </button>
	</div>

	<div id="logs"></div>
	<script>
		var log = msg => {
			document.getElementById('logs').innerHTML += msg + '<br>'
		}
		var video = document.getElementById('video');
		var button = document.getElementById('create');
		
		window.createSession = () => {
			let pc = new RTCPeerConnection({
				iceServers: [{urls: 'stun:stun.l.google.com:19302'}]
			});
			pc.oniceconnectionstatechange = e => log(pc.iceConnectionState);
			pc.onicecandidate = event => {
				if (event.candidate) { return; }
				const localDesc = JSON.stringify(pc.localDescription);
				fetch('/start', {
					method: 'POST',
					body: localDesc,
				})
					.then(resp => resp.json())
					.then(answer => pc.setRemoteDescription(answer));
			};
			pc.onnegotiationneeded = e => {
				pc.createOffer()
					.then(d => pc.setLocalDescription(d))
					.catch(log);
			};
			
			navigator.mediaDevices.getUserMedia({video: true, audio: false})
				.then(stream => {
					pc.addStream(stream);
					video.srcObject = stream;
					button.style.display = 'none';
					video.style.display = 'block';
				})
				.catch(log);
		};
	</script>
</body>
</html>`

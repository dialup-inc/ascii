package render

import (
	"os"

	"golang.org/x/sys/unix"
)

type WinSize struct {
	Rows   int
	Cols   int
	Width  int
	Height int
}

func GetWinSize() (WinSize, error) {
	ws, err := unix.IoctlGetWinsize(int(os.Stdout.Fd()), unix.TIOCGWINSZ)
	if err != nil {
		return WinSize{}, os.NewSyscallError("GetWinsize", err)
	}
	return WinSize{
		Rows:   int(ws.Row),
		Cols:   int(ws.Col),
		Width:  int(ws.Xpixel),
		Height: int(ws.Ypixel),
	}, nil
}

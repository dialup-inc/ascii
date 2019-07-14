package term

import (
	"bufio"
	"io"
	"os"

	"golang.org/x/sys/unix"
)

func makeStdinRaw() error {
	fd := int(os.Stdin.Fd())

	termios, err := unix.IoctlGetTermios(fd, ioctlReadTermios)
	if err != nil {
		return err
	}

	// This attempts to replicate the behaviour documented for cfmakeraw in
	// the termios(3) manpage.
	termios.Iflag &^= unix.IGNBRK | unix.BRKINT | unix.PARMRK | unix.ISTRIP | unix.INLCR | unix.IGNCR | unix.ICRNL | unix.IXON
	termios.Oflag &^= unix.OPOST
	termios.Lflag &^= unix.ECHO | unix.ECHONL | unix.ICANON | unix.ISIG | unix.IEXTEN
	termios.Cflag &^= unix.CSIZE | unix.PARENB
	termios.Cflag |= unix.CS8
	termios.Cc[unix.VMIN] = 1
	termios.Cc[unix.VTIME] = 0
	if err := unix.IoctlSetTermios(fd, ioctlWriteTermios, termios); err != nil {
		return err
	}

	return nil
}

func CaptureStdin(onRune func(rune)) error {
	if err := makeStdinRaw(); err != nil {
		return err
	}

	go func() {
		reader := bufio.NewReader(os.Stdin)
		for {
			r, _, err := reader.ReadRune()
			if err == io.EOF {
				break
			}
			if err != nil {
				continue
			}
			onRune(r)
		}
	}()

	return nil
}

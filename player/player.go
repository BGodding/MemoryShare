package player

import (
	"fmt"
	"math/rand"

	"github.com/DexterLB/mpvipc"
)

type Player struct {
	Conn    *mpvipc.Connection
	RunOnce bool
}

func Init() (*Player, error) {
	p := Player{}
	// TODO: Also start MPV from here?
	// MPV must be started with the command below
	// mpv --image-display-duration=inf --idle=once --keep-open=yes --input-ipc-server=/tmp/mpv_socket
	p.Conn = mpvipc.NewConnection("/tmp/mpv_socket")
	err := p.Conn.Open()
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// Will pick and play a random subset of the video file if the media length is greater than the slide duration
func (p *Player) PlayVideo(path string, mediaLength float64, slideDuration float64) error {
	clipStart := 0.0
	clipLength := mediaLength
	if mediaLength > slideDuration+1 {
		// Generate a random start pos between 0 and end - and max length
		clipStart = float64(rand.Intn(int(mediaLength - slideDuration)))
		clipLength = slideDuration
	}
	if _, err := p.Conn.Call("loadfile", fmt.Sprintf("edl://%s,start=%d,length=%d", path, int(clipStart), int(clipLength))); err != nil {
		return err
	}
	return p.Conn.Set("pause", false)
}

func (p *Player) PlayImage(path string) error {
	_, err := p.Conn.Call("loadfile", path)
	return err
}

package player

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/dexterlb/mpvipc"
	log "go.uber.org/zap"
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
func (p *Player) PlayVideoClip(path string, mediaLength float64, slideDuration float64) error {
	if p.Conn.IsClosed() {
		if err := p.Conn.Open(); err != nil {
			return err
		}
	}
	clipStart := 0.0
	if mediaLength > slideDuration+1 {
		// Generate a random start pos between 0 and end - and max length
		clipStart = float64(rand.Intn(int(mediaLength - slideDuration)))
	}
	log.S().Infof("playing video clip %#q from %fs to %fs of %fs ", path, clipStart, clipStart+slideDuration, mediaLength)
	// EDL names cannot contain  characters `,;=`
	if strings.ContainsAny(path, ",;=") {
		return fmt.Errorf("%q is an invalid path as it contains ',;='", path)

	}
	if _, err := p.Conn.Call("loadfile", fmt.Sprintf("edl://%s,start=%d,length=%d", path, int(clipStart), int(slideDuration))); err != nil {
		return err
	}
	return p.Conn.Set("pause", false)
}

func (p *Player) PlayVideo(path string) error {
	if p.Conn.IsClosed() {
		if err := p.Conn.Open(); err != nil {
			return err
		}
	}
	if _, err := p.Conn.Call("loadfile", path); err != nil {
		return err
	}
	return p.Conn.Set("pause", false)
}

func (p *Player) PlayImage(path string, slideDuration float64) error {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*time.Duration(slideDuration)+(time.Millisecond*500))
		defer cancel()
		cmd := exec.CommandContext(ctx, "feh", "-Z", "-Y", "-F", path)
		cmd.Env = append(os.Environ(), "DISPLAY=:0")
		cmd.Run()
	}()
	if p.Conn.IsClosed() {
		if err := p.Conn.Open(); err != nil {
			return err
		}
	}
	return p.Conn.Set("pause", false)
}

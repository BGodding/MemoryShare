package main

import (
	"context"
	"flag"
	"fmt"
	"memoryShare/media"
	"memoryShare/player"
	"os"
	"os/signal"
	"runtime/debug"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/DexterLB/mpvipc"
	log "go.uber.org/zap"
)

const AppVersion = "0.0.1"

type app struct {
	mediaFileHandler *media.Media
	mediaPlayer      *player.Player
	slideDuration    float64
}

func main() {
	version := flag.Bool("version", false, "prints current version and exits")
	pathString := flag.String("media-folders", ".", "comma seperated list of folders to watch")
	slideDuration := flag.Float64("media-duration", 4.0, "time for each media file to display in seconds")
	flag.Parse()

	if *version {
		buildInfo, _ := debug.ReadBuildInfo()
		fmt.Println(buildInfo.Main.Path, AppVersion, buildInfo.GoVersion)
		return
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	var wg sync.WaitGroup
	var a app
	var err error
	a.slideDuration = *slideDuration

	// Setup logging
	logger := log.Must(log.NewDevelopmentConfig().Build())
	// logger := log.Must(log.NewProductionConfig().Build())
	defer logger.Sync()

	undo := log.ReplaceGlobals(logger)
	defer undo()

	// Uncomment the line below to enable pprof
	//go func() {
	//	log.S().Info(http.ListenAndServe("localhost:6060", nil))
	//}()

	a.mediaFileHandler, err = media.Init(ctx, strings.Split(*pathString, ","))
	if err != nil {
		log.S().Fatal(err)
	}
	go func() {
		err := a.mediaFileHandler.Run(ctx)
		if err != nil {
			log.S().Error(err)
		}
	}()

	a.mediaPlayer, err = player.Init()
	if err != nil {
		log.S().Fatal(err)
	}
	defer func(Conn *mpvipc.Connection) {
		err := Conn.Close()
		if err != nil {
			log.S().Error(err)
		}
	}(a.mediaPlayer.Conn)

	// Load some initial content right away
	a.UpdateDisplay()

	// Inf loop of getting and displaying new media files
	wg.Add(1)
	go func(wg *sync.WaitGroup) {
		defer wg.Done()
		mediaTicker := time.NewTicker(time.Second * time.Duration(*slideDuration))
		for {
			select {
			case <-ctx.Done():
				return
			case <-mediaTicker.C:
				a.UpdateDisplay()
			}
		}
	}(&wg)
	// This will block until all wait group processes have called done
	wg.Wait()
}

func (a *app) UpdateDisplay() {
	for {
		// This should only return an empty file if we don't have any media to show
		file := a.mediaFileHandler.GetRandomFile()
		log.S().Debugf("%+v", file)
		if file.Path != "" {
			log.S().Debugf("playing media path %#q duration %fs", file.Path, file.MetaData.DurationSeconds)
			if file.MetaData.DurationSeconds < a.slideDuration {
				if err := a.mediaPlayer.PlayImage(file.Path); err != nil {
					log.S().Error(err)
					continue
				} else {
					return
				}
			} else if err := a.mediaPlayer.PlayVideo(file.Path, file.MetaData.DurationSeconds, a.slideDuration); err != nil {
				log.S().Error(err)
				continue
			} else {
				return
			}
		}
		time.Sleep(time.Second)
	}
}

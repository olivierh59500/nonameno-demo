// Command video exports the complete game canvas and its own audio.
package main

import (
	"flag"
	"log"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/olivierh59500/democonstructionkit/video"
	demo "nonameno-demo/dck"
)

func main() {
	config := video.Config{Output: "nonameno-demo.mp4", Title: "Nonameno", Width: 640, Height: 480, FPS: 60, TPS: 60, SampleRate: 48000, Duration: 3 * time.Minute}
	config.Flags(flag.CommandLine)
	flag.Parse()
	if err := video.Run(config, func() (ebiten.Game, error) {
		return demo.NewGame(), nil
	}); err != nil {
		log.Fatal(err)
	}
}

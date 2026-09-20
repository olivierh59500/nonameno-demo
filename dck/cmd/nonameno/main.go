package main

import (
	"log"

	"github.com/hajimehoshi/ebiten/v2"

	nonameno "nonameno-demo/dck"
)

func main() {
	ebiten.SetWindowSize(nonameno.ScreenWidth, nonameno.ScreenHeight)
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	ebiten.SetWindowTitle("NONAMENO Demo - Go/Ebitengine")
	ebiten.SetScreenClearedEveryFrame(false)

	game := nonameno.NewGame()
	defer game.Cleanup()
	if err := ebiten.RunGame(newDrawOnUpdateGame(game)); err != nil {
		log.Fatal(err)
	}
}

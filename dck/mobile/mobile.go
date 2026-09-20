// Package mobile exposes the game to ebitenmobile.
package mobile

import (
	enginemobile "github.com/hajimehoshi/ebiten/v2/mobile"

	nonameno "nonameno-demo/dck"
)

func init() {
	enginemobile.SetGame(nonameno.NewGame())
}

// Dummy forces gomobile to include this package in the Android binding.
func Dummy() {}

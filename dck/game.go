// Package nonameno implements the NONAMENO demo remake.
package nonameno

import (
	"bytes"
	kit "github.com/olivierh59500/democonstructionkit"
	"github.com/olivierh59500/democonstructionkit/motion"
	"github.com/olivierh59500/democonstructionkit/presets"
	"github.com/olivierh59500/democonstructionkit/sprites"
	"image"
	"image/color"
	originalassets "nonameno-demo"

	"github.com/olivierh59500/democonstructionkit/scrolling"
	"github.com/olivierh59500/democonstructionkit/sound"

	_ "image/png"
	"log"
	"math"
	"time"

	"github.com/hajimehoshi/ebiten/v2"

	audio "github.com/olivierh59500/democonstructionkit/sound/output"
)

const (
	ScreenWidth  = 640
	ScreenHeight = 480
	sampleRate   = 48000
)

// Embedded assets
var (
	fontData = originalassets.DCKAssetFontData()

	font8Data = originalassets.DCKAssetFont8Data()

	logoData = originalassets.DCKAssetLogoData()

	musicData = originalassets.DCKAssetMusicData()
)

// Game represents the main game state
type Game struct {
	// Images
	fontImg  *ebiten.Image
	font8Img *ebiten.Image
	logoImg  *ebiten.Image

	// Fonts
	font32 *scrolling.Atlas
	font8  *scrolling.Atlas

	// Effects
	starfield    *sprites.ProjectedField
	scrollText   *scrolling.Scrolling
	glyphPages   *motion.GlyphPageCycle
	pageRenderer *sprites.GlyphPages
	scrollTick   uint64

	// Animation state
	animationEpoch time.Time
	animationTime  float64

	// Text pages
	textPages []string

	// Audio
	audioContext *audio.Context
	audioPlayer  *audio.Player
	musicStream  *sound.Stream
	audioReady   bool
}

// NewGame creates a new game instance
func NewGame() *Game {
	now := audio.Now()
	initialPattern := int(math.Floor(math.Mod(float64(now.UnixNano()), 3)))
	g := &Game{}

	// Initialize text pages
	g.initTextPages()

	// Load images
	g.loadImages()

	// Initialize fonts
	if g.fontImg != nil {
		var err error
		g.font32, err = presets.FontAtlas("nonameno-demo", g.fontImg)
		if err != nil {
			panic(err)
		}
	}
	if g.font8Img != nil {
		var err error
		g.font8, err = presets.FontAtlas("nonameno-small", g.font8Img)
		if err != nil {
			panic(err)
		}
	}

	// Initialize effects
	starConfig, err := presets.NonamenoProjectedStars(presets.DefaultNonamenoStarsConfig())
	if err != nil {
		panic(err)
	}
	starfield, err := sprites.NewProjectedField(starConfig)
	if err != nil {
		panic(err)
	}
	g.starfield = starfield

	if g.font8 != nil {
		scrollMsg := "AS ALWAYS I DUNNO WHAT TO WRITE ON THOSE STUPID SCROLLTEXT... SO HERE IS A LITTLE GREETINGS LIST :   TOTORMAN    SINK     BADBRAIN     MURDOCK     JFAH     HEMOROIDS     TLB     TCB     ULM     OVR     MEGABUSTERS     DHS     PARADOX     REZ     RAZOR     DEMO WILL NEVER DIE ......         "
		scrollConfig, err := presets.NonamenoBottomScroll(scrollMsg, g.font8.Face(), presets.NonamenoScrollOptions{
			Width: ScreenWidth, BaselineY: 480 - 8 - 30, TicksPerSecond: 60,
			PixelsPerTick: 1, Gap: ScreenWidth + 1,
		})
		if err != nil {
			panic(err)
		}
		g.scrollText, err = scrolling.New(scrollConfig)
		if err != nil {
			panic(err)
		}
	}

	if g.font32 != nil {
		pageConfig, err := presets.NonamenoGlyphPages(g.textPages, initialPattern)
		if err != nil {
			panic(err)
		}
		g.glyphPages, err = motion.NewGlyphPageCycle(pageConfig)
		if err != nil {
			panic(err)
		}
		g.pageRenderer, err = sprites.NewGlyphPages(sprites.GlyphPagesConfig{Cycle: g.glyphPages, Font: g.font32})
		if err != nil {
			panic(err)
		}
	}

	return g
}

// initTextPages initializes the text pages
func (g *Game) initTextPages() {
	g.textPages = []string{
		"--------------------" +
			" WELCOME TO MY NEW  " +
			" NATIVE INTRO USING " +
			"    GO. I HOPE YOU  " +
			" LIKE IT. I REALLY  " +
			"LOVE THIS OLD EFFECT" +
			"IT REMEMBER ME THOSE" +
			"   OLD CRACKTROS    ",

		" AS ALWAYS I DUNNO  " +
			" WHAT TO WRITE HERE " +
			"SO I WILL SEND SOME " +
			"GREETINGS TO SINK   " +
			" TOTORMAN JFAH HMD  " +
			" REZ KHEOPS EQUINOX " +
			"TCB TLB ULM REPS TEX" +
			"DELTA FORCE WRAP....",
	}
}

// loadImages loads all image assets
func (g *Game) loadImages() {
	var err error

	// Load font image
	img, _, err := image.Decode(bytes.NewReader(fontData))
	if err == nil {
		g.fontImg = ebiten.NewImageFromImage(img)
	} else {
		log.Printf("Failed to load font: %v", err)
	}

	// Load small font image
	img, _, err = image.Decode(bytes.NewReader(font8Data))
	if err == nil {
		g.font8Img = ebiten.NewImageFromImage(img)
	} else {
		log.Printf("Failed to load font8: %v", err)
	}

	// Load logo image
	img, _, err = image.Decode(bytes.NewReader(logoData))
	if err == nil {
		g.logoImg = ebiten.NewImageFromImage(img)
	} else {
		log.Printf("Failed to load logo: %v", err)
	}
}

// initAudio initializes the audio system
func (g *Game) initAudio() {
	g.audioContext = audio.NewContext(sampleRate)

	var err error
	g.musicStream, err = sound.Open("music.ym", musicData, sound.Options{SampleRate: sampleRate, Loop: true, PCMFormat: sound.PCM16, Gain: 0.5})
	if err != nil {
		log.Printf("Failed to open music: %v", err)
		return
	}

	g.audioPlayer, err = g.audioContext.NewPlayer(g.musicStream)
	if err != nil {
		log.Printf("Failed to create audio player: %v", err)
		if closeErr := g.musicStream.Close(); closeErr != nil {
			log.Printf("Failed to close music stream: %v", closeErr)
		}
		g.musicStream = nil
		return
	}

	g.audioPlayer.Play()
}

// Update updates the game state
func (g *Game) Update() error {
	// mobile.SetGame constructs the game before Android has installed the
	// Ebitengine view. Opening the audio device is safe once Update is running.
	if !g.audioReady {
		g.audioReady = true
		g.initAudio()
	}

	now := audio.Now()
	if g.animationEpoch.IsZero() {
		g.animationEpoch = now
	}
	g.animationTime = float64(now.Sub(g.animationEpoch)) / float64(time.Millisecond)

	// Update starfield
	if g.starfield != nil {
		if err := g.starfield.Update(kit.Frame{}); err != nil {
			return err
		}
	}

	// Update scroll text
	if g.scrollText != nil {
		g.scrollTick++
		if err := g.scrollText.Update(kit.Frame{Tick: g.scrollTick, Time: float64(g.scrollTick) / 60}); err != nil {
			return err
		}
	}

	if g.glyphPages != nil {
		if err := g.glyphPages.UpdateAt(g.animationTime); err != nil {
			return err
		}
	}

	return nil
}

// Draw draws the game
func (g *Game) Draw(screen *ebiten.Image) {
	// Clear screen
	screen.Fill(color.Black)

	// Draw starfield
	if g.starfield != nil {
		g.starfield.Draw(screen)
	}

	// Draw logo
	if g.logoImg != nil {
		var op ebiten.DrawImageOptions
		op.GeoM.Translate(50, 0)
		screen.DrawImage(g.logoImg, &op)
	}

	// Draw scrolling text at bottom
	if g.scrollText != nil {
		g.scrollText.Draw(screen)
	}

	if g.pageRenderer != nil {
		g.pageRenderer.Draw(screen)
	}
}

// Layout returns the screen dimensions
func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return ScreenWidth, ScreenHeight
}

// Cleanup releases resources
func (g *Game) Cleanup() {
	if g.scrollText != nil {
		_ = g.scrollText.Close()
		g.scrollText = nil
	}
	if g.audioPlayer != nil {
		_ = g.audioPlayer.Close()
		g.audioPlayer = nil
	}
	if g.musicStream != nil {
		_ = g.musicStream.Close()
		g.musicStream = nil
	}
}

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

// ScrollText manages horizontal scrolling text with sine wave distortion
type ScrollText struct {
	renderer   *scrolling.Scrolling
	text       string
	font       *scrolling.Atlas
	scrollX    float64
	speed      float64
	textWidth  float64
	sineParams [2]SineParam
}

// SineParam represents sine wave parameters for text distortion
type SineParam struct {
	value   float64
	amp     float64
	inc     float64
	offset  float64
	sinStep float64
	cosStep float64
}

func newSineParam(value, amp, inc, offset float64) SineParam {
	sinStep, cosStep := math.Sincos(offset)
	return SineParam{
		value: value, amp: amp, inc: inc, offset: offset,
		sinStep: sinStep, cosStep: cosStep,
	}
}

func (p SineParam) advance(sine, cosine float64) (float64, float64) {
	return sine*p.cosStep + cosine*p.sinStep,
		cosine*p.cosStep - sine*p.sinStep
}

// NewScrollText creates a new scrolling text
func NewScrollText(text string, font *scrolling.Atlas, speed float64) *ScrollText {
	return &ScrollText{
		text:      text,
		font:      font,
		scrollX:   float64(ScreenWidth),
		speed:     speed,
		textWidth: float64(len(text)) * font.Metrics().LineHeight(),
		sineParams: [2]SineParam{
			newSineParam(0, 15, 0.3, 0.06),
			newSineParam(0, 15, 0.2, -0.04),
		},
	}
}

// Update updates the scroll position
func (st *ScrollText) Update() {
	st.scrollX -= st.speed

	if st.scrollX < -st.textWidth {
		st.scrollX = float64(ScreenWidth)
	}

	// Update sine parameters
	for i := range st.sineParams {
		st.sineParams[i].value += st.sineParams[i].inc
	}
}

// Draw draws the scrolling text with sine distortion
func (st *ScrollText) Draw(dst *ebiten.Image, baseY float64) {
	width := float64(st.font.Metrics().LineHeight())
	first := 0
	if st.scrollX <= -width {
		first = int(math.Floor((-width-st.scrollX)/width)) + 1
	}
	if first >= len(st.text) {
		return
	}
	if st.renderer == nil {
		images := make([]*ebiten.Image, len(st.text))
		for i := range st.text {
			images[i], _, _ = st.font.ExactGlyph(rune(st.text[i]))
		}
		var err error
		st.renderer, err = scrolling.FromImages(images, width)
		if err != nil {
			panic(err)
		}
	}
	var sines, cosines [2]float64
	for i, param := range st.sineParams {
		sines[i], cosines[i] = math.Sincos(param.value + float64(first)*param.offset)
	}
	state := scrolling.IdentityState()
	state.X = st.scrollX
	state.First = first
	state.Map = func(s scrolling.Sample, op *ebiten.DrawImageOptions) bool {
		if s.X >= float64(ScreenWidth) {
			return false
		}
		yOffset := st.sineParams[0].amp*sines[0] + st.sineParams[1].amp*sines[1]
		op.GeoM.Reset()
		op.GeoM.Translate(s.X, baseY+yOffset)
		for index, param := range st.sineParams {
			sines[index], cosines[index] = param.advance(sines[index], cosines[index])
		}
		return true
	}
	st.renderer.DrawAt(dst, state)
}

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
	scrollText   *ScrollText
	glyphPages   *motion.GlyphPageCycle
	pageRenderer *sprites.GlyphPages

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
		g.scrollText = NewScrollText(scrollMsg, g.font8, 1)
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
		g.scrollText.Update()
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
		g.scrollText.Draw(screen, 480-8-30)
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
	if g.audioPlayer != nil {
		_ = g.audioPlayer.Close()
		g.audioPlayer = nil
	}
	if g.musicStream != nil {
		_ = g.musicStream.Close()
		g.musicStream = nil
	}
}

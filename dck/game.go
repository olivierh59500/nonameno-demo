// Package nonameno implements the NONAMENO demo remake.
package nonameno

import (
	"bytes"
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
	"github.com/hajimehoshi/ebiten/v2/vector"

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

// Starfield3D represents a 3D starfield effect
type Starfield3D struct {
	stars    []Star3D
	speed    float64
	width    float64
	height   float64
	centerX  float64
	centerY  float64
	depthMax float64
}

// Star3D represents a single star in 3D space
type Star3D struct {
	x, y, z      float64
	prevX, prevY float64
	resetX       float64
	resetY       float64
}

// NewStarfield3D creates a new 3D starfield
func NewStarfield3D(numStars int, speed float64, width, height, centerX, centerY, depthMax float64) *Starfield3D {
	sf := &Starfield3D{
		speed:    speed,
		width:    width,
		height:   height,
		centerX:  centerX,
		centerY:  centerY,
		depthMax: depthMax,
		stars:    make([]Star3D, numStars),
	}

	// Initialize stars with random positions
	for i := range sf.stars {
		// Use better random distribution
		angle := float64(i) * 2.0 * math.Pi / float64(numStars)
		radius := math.Mod(float64(i*137), float64(numStars)) / float64(numStars) * width / 2
		resetAngle := math.Mod(float64(i*137), 360.0) * math.Pi / 180.0
		resetRadius := math.Mod(float64(i*89), width/2)

		sf.stars[i] = Star3D{
			x:      math.Cos(angle) * radius,
			y:      math.Sin(angle) * radius,
			z:      math.Mod(float64(i*17), depthMax),
			resetX: math.Cos(resetAngle) * resetRadius,
			resetY: math.Sin(resetAngle) * resetRadius,
		}

		// Make sure no star starts at z=0
		if sf.stars[i].z < 1 {
			sf.stars[i].z = 1
		}
	}

	return sf
}

// Update updates the starfield positions
func (sf *Starfield3D) Update() {
	for i := range sf.stars {
		// Store previous screen position for trails
		if sf.stars[i].z > 0 {
			scale := 200.0 / sf.stars[i].z
			sf.stars[i].prevX = sf.stars[i].x*scale + sf.centerX
			sf.stars[i].prevY = sf.stars[i].y*scale + sf.centerY
		}

		// Move star towards viewer
		sf.stars[i].z -= sf.speed

		// Reset star if it goes behind viewer
		if sf.stars[i].z <= 0 {
			sf.stars[i].x = sf.stars[i].resetX
			sf.stars[i].y = sf.stars[i].resetY
			sf.stars[i].z = sf.depthMax
			sf.stars[i].prevX = sf.centerX
			sf.stars[i].prevY = sf.centerY
		}
	}
}

// Draw draws the starfield to the destination image
func (sf *Starfield3D) Draw(dst *ebiten.Image) {
	for _, star := range sf.stars {
		if star.z <= 0 {
			continue
		}

		// 3D to 2D projection
		scale := 200.0 / star.z
		x := star.x*scale + sf.centerX
		y := star.y*scale + sf.centerY

		// Only draw if on screen
		if x >= 0 && x < sf.width && y >= 0 && y < sf.height {
			// Calculate star size based on depth
			size := 3.0 - (star.z / sf.depthMax * 2.5)
			if size < 0.5 {
				size = 0.5
			}

			// Draw star with varying brightness based on depth
			brightness := 1.0 - (star.z / sf.depthMax * 0.7)
			starColor := color.RGBA{
				R: uint8(255 * brightness),
				G: uint8(255 * brightness),
				B: uint8(255 * brightness),
				A: 255,
			}

			// Draw the star as a filled rectangle
			vector.FillRect(dst, float32(x-size/2), float32(y-size/2), float32(size), float32(size), starColor, false)

			// Draw motion trail for fast-moving stars
			if star.z < sf.depthMax*0.3 && star.prevX != 0 && star.prevY != 0 {
				// Draw a line from previous position to current position
				trailColor := color.RGBA{
					R: uint8(128 * brightness),
					G: uint8(128 * brightness),
					B: uint8(128 * brightness),
					A: 128,
				}
				vector.StrokeLine(dst, float32(star.prevX), float32(star.prevY), float32(x), float32(y), 1, trailColor, false)
			}
		}
	}
}

// Letter represents an animated letter (matching the original implementation structure)
type Letter struct {
	position Position
	target   Position
	ltr      int
	tweenIn  *Tween
	tweenOut *Tween
}

// Position represents a 3D position
type Position struct {
	x, y, z float64
}

type easing uint8

const (
	easeLinear easing = iota
	easeElasticOut
	easeElasticIn
)

// Tween represents an animation tween
type Tween struct {
	from       Position
	to         Position
	startTime  float64
	duration   float64
	delay      float64
	ease       easing
	onComplete func()
	active     bool
}

// NewTween creates a new tween animation
func NewTween(from, to Position, duration, delay float64, ease string, onComplete func()) *Tween {
	return newTweenAt(from, to, duration, delay, ease, onComplete, float64(time.Now().UnixMilli()))
}

func newTweenAt(from, to Position, duration, delay float64, ease string, onComplete func(), startTime float64) *Tween {
	easeType := easeLinear
	switch ease {
	case "ElasticOut":
		easeType = easeElasticOut
	case "ElasticIn":
		easeType = easeElasticIn
	}

	return &Tween{
		from:       from,
		to:         to,
		startTime:  startTime,
		duration:   duration * 1000, // Convert to milliseconds
		delay:      delay * 1000,    // Convert to milliseconds
		ease:       easeType,
		onComplete: onComplete,
		active:     true,
	}
}

// Update updates the tween and returns the current progress
func (tw *Tween) Update(pos *Position) bool {
	return tw.updateAt(pos, float64(time.Now().UnixMilli()))
}

func (tw *Tween) updateAt(pos *Position, now float64) bool {
	if !tw.active {
		return false
	}

	elapsed := now - tw.startTime - tw.delay

	if elapsed < 0 {
		return true // Still in delay
	}

	if elapsed >= tw.duration {
		pos.x = tw.to.x
		pos.y = tw.to.y
		pos.z = tw.to.z
		tw.active = false
		if tw.onComplete != nil {
			tw.onComplete()
		}
		return false
	}

	t := elapsed / tw.duration

	// Apply easing
	switch tw.ease {
	case easeElasticOut:
		t = elasticOut(t)
	case easeElasticIn:
		t = elasticIn(t)
	}

	// Interpolate
	pos.x = tw.from.x + (tw.to.x-tw.from.x)*t
	pos.y = tw.from.y + (tw.to.y-tw.from.y)*t
	pos.z = tw.from.z + (tw.to.z-tw.from.z)*t

	return true
}

// elasticOut easing function
func elasticOut(t float64) float64 {
	if t == 0 {
		return 0
	}
	if t == 1 {
		return 1
	}
	const (
		period = 0.3
		shift  = period / 4
	)
	return math.Exp2(-10*t)*math.Sin((t-shift)*2*math.Pi/period) + 1
}

// elasticIn easing function
func elasticIn(t float64) float64 {
	if t == 0 {
		return 0
	}
	if t == 1 {
		return 1
	}
	const (
		period = 0.3
		shift  = period / 4
	)
	t -= 1
	return -(math.Exp2(10*t) * math.Sin((t-shift)*2*math.Pi/period))
}

// FontChar represents a character in the font
type FontChar struct {
	x, y, width, height int
}

// BitmapFont manages bitmap font rendering
type BitmapFont struct {
	image      *ebiten.Image
	chars      map[byte]FontChar
	glyphs     [256]*ebiten.Image
	charWidth  int
	charHeight int
	tileStart  byte
}

// NewBitmapFont creates a new bitmap font
func NewBitmapFont(img *ebiten.Image, charWidth, charHeight int, tileStart byte) *BitmapFont {
	return &BitmapFont{
		image:      img,
		chars:      make(map[byte]FontChar),
		charWidth:  charWidth,
		charHeight: charHeight,
		tileStart:  tileStart,
	}
}

// InitTile initializes the font tiles
func (bf *BitmapFont) InitTile(tileWidth, tileHeight, tilesPerRow int) {
	// For 32x32 font (font.png) - 6 lines of 10 characters
	switch tileWidth {
	case 32:
		// Row 0: [NA]!"[NA][NA][NA][NA]"()
		bf.chars[byte('!')] = FontChar{x: 1 * 32, y: 0 * 32, width: 32, height: 32}
		bf.chars[byte('"')] = FontChar{x: 2 * 32, y: 0 * 32, width: 32, height: 32}
		bf.chars[byte('\'')] = FontChar{x: 7 * 32, y: 0 * 32, width: 32, height: 32} // Using " position 7
		bf.chars[byte('(')] = FontChar{x: 8 * 32, y: 0 * 32, width: 32, height: 32}
		bf.chars[byte(')')] = FontChar{x: 9 * 32, y: 0 * 32, width: 32, height: 32}

		// Row 1: [NA][NA],-.[NA]0123
		bf.chars[byte(',')] = FontChar{x: 2 * 32, y: 1 * 32, width: 32, height: 32}
		bf.chars[byte('-')] = FontChar{x: 3 * 32, y: 1 * 32, width: 32, height: 32}
		bf.chars[byte('.')] = FontChar{x: 4 * 32, y: 1 * 32, width: 32, height: 32}
		bf.chars[byte('0')] = FontChar{x: 6 * 32, y: 1 * 32, width: 32, height: 32}
		bf.chars[byte('1')] = FontChar{x: 7 * 32, y: 1 * 32, width: 32, height: 32}
		bf.chars[byte('2')] = FontChar{x: 8 * 32, y: 1 * 32, width: 32, height: 32}
		bf.chars[byte('3')] = FontChar{x: 9 * 32, y: 1 * 32, width: 32, height: 32}

		// Row 2: 456789[NA][NA][NA][NA]
		bf.chars[byte('4')] = FontChar{x: 0 * 32, y: 2 * 32, width: 32, height: 32}
		bf.chars[byte('5')] = FontChar{x: 1 * 32, y: 2 * 32, width: 32, height: 32}
		bf.chars[byte('6')] = FontChar{x: 2 * 32, y: 2 * 32, width: 32, height: 32}
		bf.chars[byte('7')] = FontChar{x: 3 * 32, y: 2 * 32, width: 32, height: 32}
		bf.chars[byte('8')] = FontChar{x: 4 * 32, y: 2 * 32, width: 32, height: 32}
		bf.chars[byte('9')] = FontChar{x: 5 * 32, y: 2 * 32, width: 32, height: 32}
		bf.chars[byte(':')] = FontChar{x: 6 * 32, y: 2 * 32, width: 32, height: 32}

		// Row 3: [NA]?[NA]ABCDEFG
		bf.chars[byte('?')] = FontChar{x: 1 * 32, y: 3 * 32, width: 32, height: 32}
		bf.chars[byte('A')] = FontChar{x: 3 * 32, y: 3 * 32, width: 32, height: 32}
		bf.chars[byte('B')] = FontChar{x: 4 * 32, y: 3 * 32, width: 32, height: 32}
		bf.chars[byte('C')] = FontChar{x: 5 * 32, y: 3 * 32, width: 32, height: 32}
		bf.chars[byte('D')] = FontChar{x: 6 * 32, y: 3 * 32, width: 32, height: 32}
		bf.chars[byte('E')] = FontChar{x: 7 * 32, y: 3 * 32, width: 32, height: 32}
		bf.chars[byte('F')] = FontChar{x: 8 * 32, y: 3 * 32, width: 32, height: 32}
		bf.chars[byte('G')] = FontChar{x: 9 * 32, y: 3 * 32, width: 32, height: 32}

		// Row 4: HIJKLMNOPQ
		bf.chars[byte('H')] = FontChar{x: 0 * 32, y: 4 * 32, width: 32, height: 32}
		bf.chars[byte('I')] = FontChar{x: 1 * 32, y: 4 * 32, width: 32, height: 32}
		bf.chars[byte('J')] = FontChar{x: 2 * 32, y: 4 * 32, width: 32, height: 32}
		bf.chars[byte('K')] = FontChar{x: 3 * 32, y: 4 * 32, width: 32, height: 32}
		bf.chars[byte('L')] = FontChar{x: 4 * 32, y: 4 * 32, width: 32, height: 32}
		bf.chars[byte('M')] = FontChar{x: 5 * 32, y: 4 * 32, width: 32, height: 32}
		bf.chars[byte('N')] = FontChar{x: 6 * 32, y: 4 * 32, width: 32, height: 32}
		bf.chars[byte('O')] = FontChar{x: 7 * 32, y: 4 * 32, width: 32, height: 32}
		bf.chars[byte('P')] = FontChar{x: 8 * 32, y: 4 * 32, width: 32, height: 32}
		bf.chars[byte('Q')] = FontChar{x: 9 * 32, y: 4 * 32, width: 32, height: 32}

		// Row 5: RSTUVWXYZ[NA]
		bf.chars[byte('R')] = FontChar{x: 0 * 32, y: 5 * 32, width: 32, height: 32}
		bf.chars[byte('S')] = FontChar{x: 1 * 32, y: 5 * 32, width: 32, height: 32}
		bf.chars[byte('T')] = FontChar{x: 2 * 32, y: 5 * 32, width: 32, height: 32}
		bf.chars[byte('U')] = FontChar{x: 3 * 32, y: 5 * 32, width: 32, height: 32}
		bf.chars[byte('V')] = FontChar{x: 4 * 32, y: 5 * 32, width: 32, height: 32}
		bf.chars[byte('W')] = FontChar{x: 5 * 32, y: 5 * 32, width: 32, height: 32}
		bf.chars[byte('X')] = FontChar{x: 6 * 32, y: 5 * 32, width: 32, height: 32}
		bf.chars[byte('Y')] = FontChar{x: 7 * 32, y: 5 * 32, width: 32, height: 32}
		bf.chars[byte('Z')] = FontChar{x: 8 * 32, y: 5 * 32, width: 32, height: 32}

		// Space character (use first position which is [NA])
		bf.chars[byte(' ')] = FontChar{x: 0 * 32, y: 0 * 32, width: 32, height: 32}
	case 8:
		// For 8x8 font (font8.png) - 2 rows of 40 characters
		// Row 0: [NA]!"[NA][NA][NA][NA]'()[NA][NA],-./0123456789:;[NA][NA][NA]?[NA]ABCDEFG
		// Row 1: HIJKLMNOPQRSTUVWXYZ[NA][NA][NA][NA][NA][NA][NA][NA][NA][NA][NA][NA][NA][NA][NA][NA][NA][NA][NA][NA][NA]

		// Row 0 characters
		bf.chars[byte('!')] = FontChar{x: 1 * 8, y: 0 * 8, width: 8, height: 8}
		bf.chars[byte('"')] = FontChar{x: 2 * 8, y: 0 * 8, width: 8, height: 8}
		bf.chars[byte('\'')] = FontChar{x: 7 * 8, y: 0 * 8, width: 8, height: 8}
		bf.chars[byte('(')] = FontChar{x: 8 * 8, y: 0 * 8, width: 8, height: 8}
		bf.chars[byte(')')] = FontChar{x: 9 * 8, y: 0 * 8, width: 8, height: 8}
		bf.chars[byte(',')] = FontChar{x: 12 * 8, y: 0 * 8, width: 8, height: 8}
		bf.chars[byte('-')] = FontChar{x: 13 * 8, y: 0 * 8, width: 8, height: 8}
		bf.chars[byte('.')] = FontChar{x: 14 * 8, y: 0 * 8, width: 8, height: 8}
		bf.chars[byte('/')] = FontChar{x: 15 * 8, y: 0 * 8, width: 8, height: 8}
		bf.chars[byte('0')] = FontChar{x: 16 * 8, y: 0 * 8, width: 8, height: 8}
		bf.chars[byte('1')] = FontChar{x: 17 * 8, y: 0 * 8, width: 8, height: 8}
		bf.chars[byte('2')] = FontChar{x: 18 * 8, y: 0 * 8, width: 8, height: 8}
		bf.chars[byte('3')] = FontChar{x: 19 * 8, y: 0 * 8, width: 8, height: 8}
		bf.chars[byte('4')] = FontChar{x: 20 * 8, y: 0 * 8, width: 8, height: 8}
		bf.chars[byte('5')] = FontChar{x: 21 * 8, y: 0 * 8, width: 8, height: 8}
		bf.chars[byte('6')] = FontChar{x: 22 * 8, y: 0 * 8, width: 8, height: 8}
		bf.chars[byte('7')] = FontChar{x: 23 * 8, y: 0 * 8, width: 8, height: 8}
		bf.chars[byte('8')] = FontChar{x: 24 * 8, y: 0 * 8, width: 8, height: 8}
		bf.chars[byte('9')] = FontChar{x: 25 * 8, y: 0 * 8, width: 8, height: 8}
		bf.chars[byte(':')] = FontChar{x: 26 * 8, y: 0 * 8, width: 8, height: 8}
		bf.chars[byte(';')] = FontChar{x: 27 * 8, y: 0 * 8, width: 8, height: 8}
		bf.chars[byte('?')] = FontChar{x: 31 * 8, y: 0 * 8, width: 8, height: 8}
		bf.chars[byte('A')] = FontChar{x: 33 * 8, y: 0 * 8, width: 8, height: 8}
		bf.chars[byte('B')] = FontChar{x: 34 * 8, y: 0 * 8, width: 8, height: 8}
		bf.chars[byte('C')] = FontChar{x: 35 * 8, y: 0 * 8, width: 8, height: 8}
		bf.chars[byte('D')] = FontChar{x: 36 * 8, y: 0 * 8, width: 8, height: 8}
		bf.chars[byte('E')] = FontChar{x: 37 * 8, y: 0 * 8, width: 8, height: 8}
		bf.chars[byte('F')] = FontChar{x: 38 * 8, y: 0 * 8, width: 8, height: 8}
		bf.chars[byte('G')] = FontChar{x: 39 * 8, y: 0 * 8, width: 8, height: 8}

		// Row 1 characters (H to Z)
		bf.chars[byte('H')] = FontChar{x: 0 * 8, y: 1 * 8, width: 8, height: 8}
		bf.chars[byte('I')] = FontChar{x: 1 * 8, y: 1 * 8, width: 8, height: 8}
		bf.chars[byte('J')] = FontChar{x: 2 * 8, y: 1 * 8, width: 8, height: 8}
		bf.chars[byte('K')] = FontChar{x: 3 * 8, y: 1 * 8, width: 8, height: 8}
		bf.chars[byte('L')] = FontChar{x: 4 * 8, y: 1 * 8, width: 8, height: 8}
		bf.chars[byte('M')] = FontChar{x: 5 * 8, y: 1 * 8, width: 8, height: 8}
		bf.chars[byte('N')] = FontChar{x: 6 * 8, y: 1 * 8, width: 8, height: 8}
		bf.chars[byte('O')] = FontChar{x: 7 * 8, y: 1 * 8, width: 8, height: 8}
		bf.chars[byte('P')] = FontChar{x: 8 * 8, y: 1 * 8, width: 8, height: 8}
		bf.chars[byte('Q')] = FontChar{x: 9 * 8, y: 1 * 8, width: 8, height: 8}
		bf.chars[byte('R')] = FontChar{x: 10 * 8, y: 1 * 8, width: 8, height: 8}
		bf.chars[byte('S')] = FontChar{x: 11 * 8, y: 1 * 8, width: 8, height: 8}
		bf.chars[byte('T')] = FontChar{x: 12 * 8, y: 1 * 8, width: 8, height: 8}
		bf.chars[byte('U')] = FontChar{x: 13 * 8, y: 1 * 8, width: 8, height: 8}
		bf.chars[byte('V')] = FontChar{x: 14 * 8, y: 1 * 8, width: 8, height: 8}
		bf.chars[byte('W')] = FontChar{x: 15 * 8, y: 1 * 8, width: 8, height: 8}
		bf.chars[byte('X')] = FontChar{x: 16 * 8, y: 1 * 8, width: 8, height: 8}
		bf.chars[byte('Y')] = FontChar{x: 17 * 8, y: 1 * 8, width: 8, height: 8}
		bf.chars[byte('Z')] = FontChar{x: 18 * 8, y: 1 * 8, width: 8, height: 8}

		// Space character (use first position which is [NA])
		bf.chars[byte(' ')] = FontChar{x: 0 * 8, y: 0 * 8, width: 8, height: 8}
	}

	for char, glyph := range bf.chars {
		rect := image.Rect(glyph.x, glyph.y, glyph.x+glyph.width, glyph.y+glyph.height)
		bf.glyphs[char] = bf.image.SubImage(rect).(*ebiten.Image)
	}
}

// DrawTile draws a single character tile (used for animated letters)
func (bf *BitmapFont) DrawTile(dst *ebiten.Image, char byte, x, y, z float64) {
	glyph := bf.glyphs[char]
	if glyph == nil {
		return
	}

	var op ebiten.DrawImageOptions
	op.GeoM.Scale(z, z)
	op.GeoM.Translate(x-float64(bf.charWidth)*z/2, y-float64(bf.charHeight)*z/2)
	dst.DrawImage(glyph, &op)
}

// ScrollText manages horizontal scrolling text with sine wave distortion
type ScrollText struct {
	renderer   *scrolling.Scrolling
	text       string
	font       *BitmapFont
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
func NewScrollText(text string, font *BitmapFont, speed float64) *ScrollText {
	return &ScrollText{
		text:      text,
		font:      font,
		scrollX:   float64(ScreenWidth),
		speed:     speed,
		textWidth: float64(len(text) * font.charWidth),
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
	width := float64(st.font.charWidth)
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
			images[i] = st.font.glyphs[st.text[i]]
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
	font32 *BitmapFont
	font8  *BitmapFont

	// Effects
	starfield   *Starfield3D
	scrollText  *ScrollText
	letters     []Letter
	letterOrder []int

	// Animation state (matching the original implementation)
	counter        int
	delayMax       int
	numPage        int
	numDelay       int
	animationEpoch time.Time
	animationTime  float64

	// Text pages
	textPages []string
	delays    [][]int

	// Audio
	audioContext *audio.Context
	audioPlayer  *audio.Player
	musicStream  *sound.Stream
	audioReady   bool
}

// NewGame creates a new game instance
func NewGame() *Game {
	now := audio.Now()
	g := &Game{
		delayMax:    3,
		numPage:     0,
		numDelay:    int(math.Floor(math.Mod(float64(now.UnixNano()), float64(3)))),
		letters:     make([]Letter, 160), // 20x8 grid
		letterOrder: make([]int, 160),
		counter:     0,
	}
	for i := range g.letterOrder {
		g.letterOrder[i] = i
	}

	// Initialize text pages
	g.initTextPages()
	g.initDelays()

	// Load images
	g.loadImages()

	// Initialize fonts
	if g.fontImg != nil {
		g.font32 = NewBitmapFont(g.fontImg, 32, 32, 32)
		g.font32.InitTile(32, 32, 10)
	}
	if g.font8Img != nil {
		g.font8 = NewBitmapFont(g.font8Img, 8, 8, 32)
		g.font8.InitTile(8, 8, 10)
	}

	// Initialize effects
	g.starfield = NewStarfield3D(500, 2, 640, 480, 320, 240, 100)

	if g.font8 != nil {
		scrollMsg := "AS ALWAYS I DUNNO WHAT TO WRITE ON THOSE STUPID SCROLLTEXT... SO HERE IS A LITTLE GREETINGS LIST :   TOTORMAN    SINK     BADBRAIN     MURDOCK     JFAH     HEMOROIDS     TLB     TCB     ULM     OVR     MEGABUSTERS     DHS     PARADOX     REZ     RAZOR     DEMO WILL NEVER DIE ......         "
		g.scrollText = NewScrollText(scrollMsg, g.font8, 1)
	}

	// Initialize letters (matching the original implementation init)
	g.initLetters()

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

// initDelays initializes the delay patterns
func (g *Game) initDelays() {
	g.delays = [][]int{
		// Pattern 0 - Wave from corners
		{
			0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19,
			39, 38, 37, 36, 35, 34, 33, 32, 31, 30, 29, 28, 27, 26, 25, 24, 23, 22, 21, 20,
			40, 41, 42, 43, 44, 45, 46, 47, 48, 49, 50, 51, 52, 53, 54, 55, 56, 57, 58, 59,
			79, 78, 77, 76, 75, 74, 73, 72, 71, 70, 69, 68, 67, 66, 65, 64, 63, 62, 61, 60,
			80, 81, 82, 83, 84, 85, 86, 87, 88, 89, 90, 91, 92, 93, 94, 95, 96, 97, 98, 99,
			119, 118, 117, 116, 115, 114, 113, 112, 111, 110, 109, 108, 107, 106, 105, 104, 103, 102, 101, 100,
			120, 121, 122, 123, 124, 125, 126, 127, 128, 129, 130, 131, 132, 133, 134, 135, 136, 137, 138, 139,
			159, 158, 157, 156, 155, 154, 153, 152, 151, 150, 149, 148, 147, 146, 145, 144, 143, 142, 141, 140,
		},
		// Pattern 1 - Vertical stripes
		{
			0, 15, 16, 31, 32, 47, 48, 63, 64, 79, 80, 95, 96, 111, 112, 127, 128, 143, 144, 159,
			1, 14, 17, 30, 33, 46, 49, 62, 65, 78, 81, 94, 97, 110, 113, 126, 129, 142, 145, 158,
			2, 13, 18, 29, 34, 45, 50, 61, 66, 77, 82, 93, 98, 109, 114, 125, 130, 141, 146, 157,
			3, 12, 19, 28, 35, 44, 51, 60, 67, 76, 83, 92, 99, 108, 115, 124, 131, 140, 147, 156,
			4, 11, 20, 27, 36, 43, 52, 59, 68, 75, 84, 91, 100, 107, 116, 123, 132, 139, 148, 155,
			5, 10, 21, 26, 37, 42, 53, 58, 69, 74, 85, 90, 101, 106, 117, 122, 133, 138, 149, 154,
			6, 9, 22, 25, 38, 41, 54, 57, 70, 73, 86, 89, 102, 105, 118, 121, 134, 137, 150, 153,
			7, 8, 23, 24, 39, 40, 55, 56, 71, 72, 87, 88, 103, 104, 119, 120, 135, 136, 151, 152,
		},
		// Pattern 2 - Spiral
		{
			0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19,
			51, 52, 53, 54, 55, 56, 57, 58, 59, 60, 61, 62, 63, 64, 65, 66, 67, 68, 69, 20,
			50, 95, 96, 97, 98, 99, 100, 101, 102, 103, 104, 105, 106, 107, 108, 109, 110, 111, 70, 21,
			49, 94, 131, 132, 133, 134, 135, 136, 137, 138, 138, 140, 141, 142, 143, 144, 145, 112, 71, 22,
			48, 93, 130, 159, 158, 157, 156, 155, 154, 153, 152, 151, 150, 149, 148, 147, 146, 113, 72, 23,
			47, 92, 129, 128, 127, 126, 125, 124, 123, 122, 121, 120, 119, 118, 117, 116, 115, 114, 73, 24,
			46, 91, 90, 89, 88, 87, 86, 85, 84, 83, 82, 81, 80, 79, 78, 77, 76, 75, 74, 25,
			45, 44, 43, 42, 41, 40, 39, 38, 37, 36, 35, 34, 33, 32, 31, 30, 29, 28, 27, 26,
		},
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

// initLetters initializes the letter animations (matching the original implementation logic)
func (g *Game) newTween(from, to Position, duration, delay float64, ease string, onComplete func()) *Tween {
	return newTweenAt(from, to, duration, delay, ease, onComplete, g.animationTime)
}

func (g *Game) initLetters() {
	if g.font32 == nil {
		return
	}

	num := 0
	for j := 0; j < 8; j++ {
		for i := 0; i < 20; i++ {
			g.letters[num] = Letter{
				ltr: int(g.textPages[g.numPage][num]) - int(g.font32.tileStart),
				position: Position{
					x: 320,
					y: 240,
					z: 0.000000001,
				},
				target: Position{
					x: float64(16 + i*32),
					y: float64(16 + 32*5 + j*32 - 16),
					z: 1.0,
				},
			}

			// Create initial tween
			delay := float64(g.delays[g.numDelay][num]) * 0.04
			g.letters[num].tweenIn = g.newTween(
				g.letters[num].position,
				g.letters[num].target,
				2, // 2 seconds duration
				delay,
				"ElasticOut",
				g.countMeIn,
			)

			num++
		}
	}
}

// countMeIn is called when a letter animation completes
func (g *Game) countMeIn() {
	g.counter++
	if g.counter == 160 {
		// All letters have arrived, prepare reverse animation
		g.numDelay = (g.numDelay + 1) % g.delayMax

		num := 0
		for j := 0; j < 8; j++ {
			for i := 0; i < 20; i++ {
				delay := float64(g.delays[g.numDelay][num]) * 0.04
				g.letters[num].tweenOut = g.newTween(
					g.letters[num].position,                  // Current position
					Position{x: 320, y: 240, z: 0.000000001}, // Back to center
					1,                                        // 1 second duration
					delay,
					"ElasticIn",
					g.countMeOut,
				)
				num++
			}
		}
		g.counter = 0
	}
}

// countMeOut is called when a letter reverse animation completes
func (g *Game) countMeOut() {
	g.counter++
	if g.counter == 160 {
		// All letters have left, change page
		g.numPage = (g.numPage + 1) % len(g.textPages)
		g.numDelay = (g.numDelay + 1) % g.delayMax

		num := 0
		for j := 0; j < 8; j++ {
			for i := 0; i < 20; i++ {
				g.letters[num].ltr = int(g.textPages[g.numPage][num]) - int(g.font32.tileStart)
				g.letters[num].position = Position{x: 320, y: 240, z: 0.000000001}
				g.letters[num].target = Position{
					x: float64(16 + i*32),
					y: float64(16 + 32*5 + j*32 - 16),
					z: 1.0,
				}

				delay := float64(g.delays[g.numDelay][num]) * 0.04
				g.letters[num].tweenIn = g.newTween(
					g.letters[num].position,
					g.letters[num].target,
					2, // 2 seconds
					delay,
					"ElasticOut",
					g.countMeIn,
				)
				g.letters[num].tweenOut = nil // Clear old tween
				num++
			}
		}
		g.counter = 0
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
		g.starfield.Update()
	}

	// Update scroll text
	if g.scrollText != nil {
		g.scrollText.Update()
	}

	// Update letter animations
	for i := range g.letters {
		if g.letters[i].tweenIn != nil && g.letters[i].tweenIn.active {
			g.letters[i].tweenIn.updateAt(&g.letters[i].position, g.animationTime)
		}
		if g.letters[i].tweenOut != nil && g.letters[i].tweenOut.active {
			g.letters[i].tweenOut.updateAt(&g.letters[i].position, g.animationTime)
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

	// Sort and draw letters by Z depth
	g.drawLetters(screen)
}

// drawLetters draws the animated letters sorted by depth
func (g *Game) drawLetters(screen *ebiten.Image) {
	if g.font32 == nil {
		return
	}

	g.sortLettersByDepth()

	for _, idx := range g.letterOrder {
		letter := &g.letters[idx]
		if letter.position.z > 0 {
			char := byte(letter.ltr + int(g.font32.tileStart))
			g.font32.DrawTile(screen, char, letter.position.x, letter.position.y, letter.position.z)
		}
	}
}

func (g *Game) sortLettersByDepth() {
	// The order changes gradually between frames, so an in-place insertion sort
	// is both allocation-free and faster than rebuilding and sorting a slice.
	for i := 1; i < len(g.letterOrder); i++ {
		index := g.letterOrder[i]
		depth := g.letters[index].position.z
		j := i
		for j > 0 && g.letters[g.letterOrder[j-1]].position.z > depth {
			g.letterOrder[j] = g.letterOrder[j-1]
			j--
		}
		g.letterOrder[j] = index
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

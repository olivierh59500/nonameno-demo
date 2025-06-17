package main

import (
	"bytes"
	_ "embed"
	"fmt"
	"image"
	"image/color"
	_ "image/png"
	"io"
	"log"
	"math"
	"sync"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/olivierh59500/ym-player/pkg/stsound"
)

const (
	screenWidth  = 640
	screenHeight = 480
	sampleRate   = 44100
)

// Embedded assets
var (
	//go:embed assets/font.png
	fontData []byte
	//go:embed assets/font8.png
	font8Data []byte
	//go:embed assets/logo.png
	logoData []byte
	//go:embed assets/music.ym
	musicData []byte
)

// YMPlayer wraps the YM player for Ebiten audio
type YMPlayer struct {
	player       *stsound.StSound
	sampleRate   int
	buffer       []int16
	mutex        sync.Mutex
	position     int64
	totalSamples int64
	loop         bool
	volume       float64
}

// NewYMPlayer creates a new YM player instance
func NewYMPlayer(data []byte, sampleRate int, loop bool) (*YMPlayer, error) {
	player := stsound.CreateWithRate(sampleRate)

	if err := player.LoadMemory(data); err != nil {
		player.Destroy()
		return nil, fmt.Errorf("failed to load YM data: %w", err)
	}

	player.SetLoopMode(loop)

	info := player.GetInfo()
	totalSamples := int64(info.MusicTimeInMs) * int64(sampleRate) / 1000

	return &YMPlayer{
		player:       player,
		sampleRate:   sampleRate,
		buffer:       make([]int16, 4096),
		totalSamples: totalSamples,
		loop:         loop,
		volume:       0.7,
	}, nil
}

// Read implements io.Reader for audio streaming
func (y *YMPlayer) Read(p []byte) (n int, err error) {
	y.mutex.Lock()
	defer y.mutex.Unlock()

	samplesNeeded := len(p) / 4
	outBuffer := make([]int16, samplesNeeded*2)

	processed := 0
	for processed < samplesNeeded {
		chunkSize := samplesNeeded - processed
		if chunkSize > len(y.buffer) {
			chunkSize = len(y.buffer)
		}

		if !y.player.Compute(y.buffer[:chunkSize], chunkSize) {
			if !y.loop {
				for i := processed * 2; i < len(outBuffer); i++ {
					outBuffer[i] = 0
				}
				err = io.EOF
				break
			}
		}

		for i := 0; i < chunkSize; i++ {
			sample := int16(float64(y.buffer[i]) * y.volume)
			outBuffer[(processed+i)*2] = sample
			outBuffer[(processed+i)*2+1] = sample
		}

		processed += chunkSize
		y.position += int64(chunkSize)
	}

	buf := make([]byte, 0, len(outBuffer)*2)
	for _, sample := range outBuffer {
		buf = append(buf, byte(sample), byte(sample>>8))
	}

	copy(p, buf)
	n = len(buf)
	if n > len(p) {
		n = len(p)
	}

	return n, err
}

// Seek implements io.Seeker
func (y *YMPlayer) Seek(offset int64, whence int) (int64, error) {
	y.mutex.Lock()
	defer y.mutex.Unlock()

	var newPos int64
	switch whence {
	case io.SeekStart:
		newPos = offset
	case io.SeekCurrent:
		newPos = y.position + offset
	case io.SeekEnd:
		newPos = y.totalSamples + offset
	default:
		return 0, fmt.Errorf("invalid whence: %d", whence)
	}

	if newPos < 0 {
		newPos = 0
	}
	if newPos > y.totalSamples {
		newPos = y.totalSamples
	}

	y.position = newPos
	return newPos, nil
}

// Close releases resources
func (y *YMPlayer) Close() error {
	y.mutex.Lock()
	defer y.mutex.Unlock()

	if y.player != nil {
		y.player.Destroy()
		y.player = nil
	}
	return nil
}

// Starfield3D represents a 3D starfield effect
type Starfield3D struct {
	stars    []Star3D
	numStars int
	speed    float64
	width    float64
	height   float64
	centerX  float64
	centerY  float64
	depthMax float64
	color    color.Color
}

// Star3D represents a single star in 3D space
type Star3D struct {
	x, y, z float64
	prevX   float64
	prevY   float64
}

// NewStarfield3D creates a new 3D starfield
func NewStarfield3D(numStars int, speed float64, width, height, centerX, centerY, depthMax float64) *Starfield3D {
	sf := &Starfield3D{
		numStars: numStars,
		speed:    speed,
		width:    width,
		height:   height,
		centerX:  centerX,
		centerY:  centerY,
		depthMax: depthMax,
		color:    color.White,
		stars:    make([]Star3D, numStars),
	}

	// Initialize stars with random positions
	for i := range sf.stars {
		// Use better random distribution
		angle := float64(i) * 2.0 * math.Pi / float64(numStars)
		radius := math.Mod(float64(i*137), float64(numStars)) / float64(numStars) * width / 2

		sf.stars[i] = Star3D{
			x: math.Cos(angle) * radius,
			y: math.Sin(angle) * radius,
			z: math.Mod(float64(i*17), depthMax),
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
			// Create new star at far distance
			angle := math.Mod(float64(i*137), 360.0) * math.Pi / 180.0
			radius := math.Mod(float64(i*89), sf.width/2)

			sf.stars[i].x = math.Cos(angle) * radius
			sf.stars[i].y = math.Sin(angle) * radius
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
			ebitenutil.DrawRect(dst, x-size/2, y-size/2, size, size, starColor)

			// Draw motion trail for fast-moving stars
			if star.z < sf.depthMax*0.3 && star.prevX != 0 && star.prevY != 0 {
				// Draw a line from previous position to current position
				trailColor := color.RGBA{
					R: uint8(128 * brightness),
					G: uint8(128 * brightness),
					B: uint8(128 * brightness),
					A: 128,
				}
				ebitenutil.DrawLine(dst, star.prevX, star.prevY, x, y, trailColor)
			}
		}
	}
}

// Letter represents an animated letter (matching JavaScript structure)
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

// Tween represents an animation tween
type Tween struct {
	from       Position
	to         Position
	startTime  float64
	duration   float64
	delay      float64
	ease       string
	onComplete func()
	active     bool
}

// NewTween creates a new tween animation
func NewTween(from, to Position, duration, delay float64, ease string, onComplete func()) *Tween {
	return &Tween{
		from:       from,
		to:         to,
		startTime:  float64(time.Now().UnixMilli()),
		duration:   duration * 1000, // Convert to milliseconds
		delay:      delay * 1000,    // Convert to milliseconds
		ease:       ease,
		onComplete: onComplete,
		active:     true,
	}
}

// Update updates the tween and returns the current progress
func (tw *Tween) Update(pos *Position) bool {
	if !tw.active {
		return false
	}

	now := float64(time.Now().UnixMilli())
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
	if tw.ease == "ElasticOut" {
		t = elasticOut(t)
	} else if tw.ease == "ElasticIn" {
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
	p := 0.3
	s := p / 4
	return math.Pow(2, -10*t)*math.Sin((t-s)*2*math.Pi/p) + 1
}

// elasticIn easing function
func elasticIn(t float64) float64 {
	if t == 0 {
		return 0
	}
	if t == 1 {
		return 1
	}
	p := 0.3
	s := p / 4
	t -= 1
	return -(math.Pow(2, 10*t) * math.Sin((t-s)*2*math.Pi/p))
}

// FontChar represents a character in the font
type FontChar struct {
	x, y, width, height int
}

// BitmapFont manages bitmap font rendering
type BitmapFont struct {
	image      *ebiten.Image
	chars      map[byte]FontChar
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
	if tileWidth == 32 {
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
	} else if tileWidth == 8 {
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
}

// DrawTile draws a single character tile (used for animated letters)
func (bf *BitmapFont) DrawTile(dst *ebiten.Image, char byte, x, y, z float64) {
	fc, ok := bf.chars[char]
	if !ok {
		return
	}

	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(z, z)
	op.GeoM.Translate(x-float64(fc.width)*z/2, y-float64(fc.height)*z/2)

	srcRect := image.Rect(fc.x, fc.y, fc.x+fc.width, fc.y+fc.height)
	dst.DrawImage(bf.image.SubImage(srcRect).(*ebiten.Image), op)
}

// ScrollText manages horizontal scrolling text with sine wave distortion
type ScrollText struct {
	text       string
	font       *BitmapFont
	scrollX    float64
	speed      float64
	sineParams []SineParam
}

// SineParam represents sine wave parameters for text distortion
type SineParam struct {
	value  float64
	amp    float64
	inc    float64
	offset float64
}

// NewScrollText creates a new scrolling text
func NewScrollText(text string, font *BitmapFont, speed float64) *ScrollText {
	return &ScrollText{
		text:    text,
		font:    font,
		scrollX: float64(screenWidth),
		speed:   speed,
		sineParams: []SineParam{
			{value: 0, amp: 15, inc: 0.3, offset: 0.06},
			{value: 0, amp: 15, inc: 0.2, offset: -0.04},
		},
	}
}

// Update updates the scroll position
func (st *ScrollText) Update() {
	st.scrollX -= st.speed

	// Calculate text width
	textWidth := len(st.text) * st.font.charWidth
	if st.scrollX < -float64(textWidth) {
		st.scrollX = float64(screenWidth)
	}

	// Update sine parameters
	for i := range st.sineParams {
		st.sineParams[i].value += st.sineParams[i].inc
	}
}

// Draw draws the scrolling text with sine distortion
func (st *ScrollText) Draw(dst *ebiten.Image, baseY float64) {
	// Draw each character with sine wave distortion
	for i, char := range st.text {
		x := st.scrollX + float64(i*st.font.charWidth)

		// Only draw if character is on screen
		if x > -float64(st.font.charWidth) && x < float64(screenWidth) {
			// Apply sine wave distortion to Y position
			yOffset := 0.0
			for _, param := range st.sineParams {
				yOffset += param.amp * math.Sin(param.value+float64(i)*param.offset)
			}

			y := baseY + yOffset

			// Draw the character
			if fc, ok := st.font.chars[byte(char)]; ok {
				op := &ebiten.DrawImageOptions{}
				op.GeoM.Translate(x, y)

				srcRect := image.Rect(fc.x, fc.y, fc.x+fc.width, fc.y+fc.height)
				dst.DrawImage(st.font.image.SubImage(srcRect).(*ebiten.Image), op)
			}
		}
	}
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
	starfield  *Starfield3D
	scrollText *ScrollText
	letters    []Letter

	// Animation state (matching JavaScript)
	counter    int
	pageMax    int
	delayMax   int
	numPage    int
	numDelay   int
	frameCount int

	// Text pages
	textPages []string
	delays    [][]int

	// Audio
	audioContext *audio.Context
	audioPlayer  *audio.Player
	ymPlayer     *YMPlayer
}

// NewGame creates a new game instance
func NewGame() *Game {
	g := &Game{
		pageMax:  1,
		delayMax: 3,
		numPage:  0,
		numDelay: int(math.Floor(math.Mod(float64(time.Now().UnixNano()), float64(3)))),
		letters:  make([]Letter, 160), // 20x8 grid
		counter:  0,
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

	// Initialize letters (matching JavaScript init)
	g.initLetters()

	// Initialize audio
	g.initAudio()

	return g
}

// initTextPages initializes the text pages
func (g *Game) initTextPages() {
	g.textPages = []string{
		"--------------------" +
			" WELCOME TO MY NEW  " +
			" CANVAS INTRO USING " +
			" CODEF. I HOPE YOU  " +
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

// initLetters initializes the letter animations (matching JavaScript logic)
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
			g.letters[num].tweenIn = NewTween(
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
				g.letters[num].tweenOut = NewTween(
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
				g.letters[num].tweenIn = NewTween(
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
	g.ymPlayer, err = NewYMPlayer(musicData, sampleRate, true)
	if err != nil {
		log.Printf("Failed to create YM player: %v", err)
		return
	}

	g.audioPlayer, err = g.audioContext.NewPlayer(g.ymPlayer)
	if err != nil {
		log.Printf("Failed to create audio player: %v", err)
		g.ymPlayer.Close()
		g.ymPlayer = nil
		return
	}

	g.audioPlayer.SetVolume(0.7)
	g.audioPlayer.Play()
}

// Update updates the game state
func (g *Game) Update() error {
	g.frameCount++

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
		// Update incoming animation
		if g.letters[i].tweenIn != nil && g.letters[i].tweenIn.active {
			g.letters[i].tweenIn.Update(&g.letters[i].position)
		}
		// Update outgoing animation
		if g.letters[i].tweenOut != nil && g.letters[i].tweenOut.active {
			g.letters[i].tweenOut.Update(&g.letters[i].position)
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
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(50, 0)
		screen.DrawImage(g.logoImg, op)
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

	// Create a sorted index by Z position
	indices := make([]int, len(g.letters))
	for i := range indices {
		indices[i] = i
	}

	// Sort indices by letter Z position (back to front)
	for i := 0; i < len(indices)-1; i++ {
		for j := i + 1; j < len(indices); j++ {
			if g.letters[indices[i]].position.z > g.letters[indices[j]].position.z {
				indices[i], indices[j] = indices[j], indices[i]
			}
		}
	}

	// Draw letters in sorted order
	for _, idx := range indices {
		letter := &g.letters[idx]
		// Always draw letters with positive Z
		if letter.position.z > 0 {
			char := byte(letter.ltr + int(g.font32.tileStart))
			g.font32.DrawTile(screen, char, letter.position.x, letter.position.y, letter.position.z)
		}
	}
}

// Layout returns the screen dimensions
func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return screenWidth, screenHeight
}

// Cleanup releases resources
func (g *Game) Cleanup() {
	if g.audioPlayer != nil {
		g.audioPlayer.Close()
	}
	if g.ymPlayer != nil {
		g.ymPlayer.Close()
	}
}

func main() {
	ebiten.SetWindowSize(screenWidth, screenHeight)
	ebiten.SetWindowTitle("NONAMENO Demo - Go/Ebiten Conversion")

	game := NewGame()

	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}

	game.Cleanup()
}

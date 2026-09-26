package nonameno

import (
	"encoding/binary"
	"errors"
	"io"
	"math"
	"testing"

	"github.com/olivierh59500/democonstructionkit/motion"
	"github.com/olivierh59500/democonstructionkit/presets"
	"github.com/olivierh59500/democonstructionkit/sound"
)

func TestMusicStreamReadProducesStereoWithoutAllocating(t *testing.T) {
	player, err := sound.Open("music.ym", musicData, sound.Options{SampleRate: sampleRate, Loop: true, PCMFormat: sound.PCM16, Gain: 0.5})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := player.Close(); err != nil {
			t.Errorf("Close: %v", err)
		}
	})

	buffer := make([]byte, 4096*4)
	read := func() {
		n, err := player.Read(buffer)
		if err != nil {
			t.Fatalf("Read: %v", err)
		}
		if n != len(buffer) {
			t.Fatalf("Read bytes = %d, want %d", n, len(buffer))
		}
	}

	read()
	for i := 0; i < len(buffer); i += 4 {
		left := binary.LittleEndian.Uint16(buffer[i : i+2])
		right := binary.LittleEndian.Uint16(buffer[i+2 : i+4])
		if left != right {
			t.Fatalf("frame %d is not mono duplicated to stereo: %d != %d", i/4, left, right)
		}
	}

	if allocations := testing.AllocsPerRun(20, read); allocations != 0 {
		t.Fatalf("Read allocations = %v, want 0", allocations)
	}
}

func TestMusicStreamCloseIsIdempotent(t *testing.T) {
	player, err := sound.Open("music.ym", musicData, sound.Options{SampleRate: sampleRate, Loop: true, PCMFormat: sound.PCM16, Gain: 0.5})
	if err != nil {
		t.Fatal(err)
	}
	if err := player.Close(); err != nil {
		t.Fatal(err)
	}
	if err := player.Close(); err != nil {
		t.Fatal(err)
	}

	if _, err := player.Read(make([]byte, 4)); !errors.Is(err, io.ErrClosedPipe) {
		t.Fatalf("Read after Close error = %v, want %v", err, io.ErrClosedPipe)
	}
}

func TestAuthoredPagesUseReusableGlyphCycle(t *testing.T) {
	game := &Game{}
	game.initTextPages()
	config, err := presets.NonamenoGlyphPages(game.textPages, 0)
	if err != nil {
		t.Fatal(err)
	}
	cycle, err := motion.NewGlyphPageCycle(config)
	if err != nil {
		t.Fatal(err)
	}
	if cycle.Count() != 160 || cycle.Glyph(0).Rune != '-' {
		t.Fatalf("authored page was not mapped to the 20-by-8 grid")
	}
}

func TestBottomScrollWavesMatchAuthoredPhases(t *testing.T) {
	waves := motion.Waves(presets.NonamenoBottomWaves(8, 60))
	for tick := 0; tick < 1000; tick += 7 {
		for index := 0; index < 80; index++ {
			want := 15*math.Sin(float64(tick)*.3+float64(index)*.06) +
				15*math.Sin(float64(tick)*.2-float64(index)*.04)
			got := waves.At(float64(index*8), float64(tick)/60)
			if math.Abs(got-want) > 1e-12 {
				t.Fatalf("tick %d glyph %d: %.12f != %.12f", tick, index, got, want)
			}
		}
	}
}

func BenchmarkMusicStreamRead4096(b *testing.B) {
	player, err := sound.Open("music.ym", musicData, sound.Options{SampleRate: sampleRate, Loop: true, PCMFormat: sound.PCM16, Gain: 0.5})
	if err != nil {
		b.Fatal(err)
	}
	b.Cleanup(func() { _ = player.Close() })

	buffer := make([]byte, 4096*4)
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		if _, err := player.Read(buffer); err != nil {
			b.Fatal(err)
		}
	}
}

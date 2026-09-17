package nonameno

import (
	"encoding/binary"
	"errors"
	"io"
	"math"
	"testing"
)

var benchmarkPosition Position

func TestYMPlayerReadProducesStereoWithoutAllocating(t *testing.T) {
	player, err := NewYMPlayer(musicData, sampleRate, true)
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

func TestYMPlayerCloseIsIdempotent(t *testing.T) {
	player, err := NewYMPlayer(musicData, sampleRate, true)
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

func TestSortLettersByDepth(t *testing.T) {
	game := &Game{
		letters:     make([]Letter, 160),
		letterOrder: make([]int, 160),
	}
	for i := range game.letters {
		game.letters[i].position.z = float64((i * 73) % len(game.letters))
		game.letterOrder[i] = i
	}

	game.sortLettersByDepth()
	for i := 1; i < len(game.letterOrder); i++ {
		previous := game.letters[game.letterOrder[i-1]].position.z
		current := game.letters[game.letterOrder[i]].position.z
		if previous > current {
			t.Fatalf("depths at %d are not sorted: %v > %v", i, previous, current)
		}
	}
	if allocations := testing.AllocsPerRun(20, game.sortLettersByDepth); allocations != 0 {
		t.Fatalf("sort allocations = %v, want 0", allocations)
	}
}

func TestSineParamRecurrenceMatchesDirectCalculation(t *testing.T) {
	param := newSineParam(0.37, 15, 0.3, -0.04)
	sine, cosine := math.Sincos(param.value)
	for i := 0; i < 1000; i++ {
		want := math.Sin(param.value + float64(i)*param.offset)
		if difference := math.Abs(sine - want); difference > 1e-12 {
			t.Fatalf("sine %d differs by %g", i, difference)
		}
		sine, cosine = param.advance(sine, cosine)
	}
}

func BenchmarkYMPlayerRead4096(b *testing.B) {
	player, err := NewYMPlayer(musicData, sampleRate, true)
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

func BenchmarkTweenUpdateFrame(b *testing.B) {
	position := Position{}
	tweens := make([]*Tween, 160)
	for i := range tweens {
		tweens[i] = newTweenAt(Position{}, Position{x: 1, y: 1, z: 1}, 1e12, 0, "ElasticOut", nil, 0)
	}

	now := 0.0
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		now += 16.0
		for _, tween := range tweens {
			tween.updateAt(&position, now)
		}
	}
	benchmarkPosition = position
}

func BenchmarkLetterDepthSort(b *testing.B) {
	game := &Game{
		letters:     make([]Letter, 160),
		letterOrder: make([]int, 160),
	}
	for i := range game.letters {
		game.letters[i].position.z = float64((i * 73) % len(game.letters))
		game.letterOrder[i] = i
	}
	game.sortLettersByDepth()

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		game.sortLettersByDepth()
	}
}

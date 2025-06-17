# NONAMENO Demo - Go/Ebiten Implementation

A faithful Go/Ebiten port of one of NONAMENO demo originally created using the NATIVE the original implementation framework. This implementation recreates the iconic scrolling text effects, 3D starfield, and animated letter transitions while maintaining the nostalgic feel of the original Atari ST demo scene.

## Features

- **3D Starfield**: Classic perspective-correct starfield animation
- **Animated Letters**: Elastic tweening animations for text transitions
- **Scrolling Text**: Horizontal scrolltext with sine wave distortion
- **YM Music Playback**: Authentic chiptune music using YM format
- **Bitmap Font Rendering**: Original demo fonts with proper character mapping
- **Frame-Perfect Timing**: Smooth 60 FPS rendering

## Requirements

- Go 1.19 or higher
- [Ebiten v2](https://github.com/hajimehoshi/ebiten) game engine
- [ym-player](https://github.com/olivierh59500/ym-player) for YM music playback

## Installation

```bash
# Clone the repository
git clone https://github.com/olivierh59500/nonameno-demo
cd nonameno-demo

# Download dependencies
go mod init nonameno-demo
go get github.com/hajimehoshi/ebiten/v2
go get github.com/olivierh59500/ym-player

# Build and run
go run main.go
```

## Assets Structure

Place the following assets in the `assets/` directory:
- `font.png` - Main 32x32 bitmap font
- `font8.png` - Small 8x8 bitmap font for scrolltext
- `logo.png` - TCB logo graphic
- `music.ym` - YM format chiptune music

## Technical Details

### Font System
The demo uses a custom bitmap font system that maps ASCII characters to sprite tiles. Three different font sizes are supported:
- 32x32 pixels for main animated text
- 8x8 pixels for scrolling text
- Character mapping follows the original NATIVE demo layout

### Animation System
Letter animations use custom tweening with elastic easing functions:
- **ElasticOut**: For letters appearing on screen
- **ElasticIn**: For letters disappearing
- Multiple delay patterns create wave effects

### Audio System
YM music playback is handled through a custom wrapper that:
- Provides io.Reader interface for Ebiten audio
- Supports looping playback
- Maintains proper sample rate conversion

### Effects Pipeline
1. **3D Starfield** - Rendered first as background
2. **Logo** - Static position at top
3. **Animated Letters** - Sorted by Z-depth for proper overlap
4. **Scrolltext** - Rendered last at bottom with sine distortion

## Code Structure

```
main.go
├── YMPlayer        - Audio playback wrapper
├── Starfield3D     - 3D starfield effect
├── Letter          - Animated letter structure
├── Vec3            - 3D vector for positioning
├── Tween           - Animation tweening system
├── BitmapFont      - Font rendering system
├── ScrollText      - Horizontal scrolling text
└── Game            - Main game loop and state
```

## Performance Considerations

- All sprites are pre-loaded into memory
- Font characters are cached as sub-images
- Minimal allocations during render loop
- Efficient depth sorting for letter overlap

## Original Credits

This is a port of the NONAMENO demo originally created by:
- **Code**: NONAMENO
- **Graphics**: ???
- **Original Platform**: Web
- **Framework**: NATIVE

## License

This port maintains the spirit of the demo scene - share, learn, and create. The original demo was created for the love of the art form, and this port continues that tradition.

## Contributing

Feel free to submit issues or pull requests if you find bugs or want to add features while maintaining the authenticity of the original demo.

## Acknowledgments

Special thanks to the Atari ST demo scene community for preserving these pieces of digital art history. The dedication to pushing hardware limits and creating beauty within constraints continues to inspire developers today.
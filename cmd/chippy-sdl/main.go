// -build js,tty

/*
Copyright (C) 2016 Andreas T Jonsson

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with this program.  If not, see <http://www.gnu.org/licenses/>.
*/

package main

/*
#cgo LDFLAGS: -lm
#include <math.h>
void audioCallback(void *userdata, unsigned char *stream, int len) {
	int i;
	for (i = 0; i < len; i++) {
		stream[i] = (sin(i / 4) + 1) * 128;
	}
}
*/
import "C"

import (
	"flag"
	"fmt"
	"image/color"
	"io/ioutil"
	"math/rand"
	"os"
	"path"
	"runtime"
	"time"
	"unsafe"

	"github.com/andreas-jonsson/chip8/chip8"
	"github.com/veandco/go-sdl2/sdl"
)

var keymap = [16]string{
	"X",
	"1",
	"2",
	"3",
	"Q",
	"W",
	"E",
	"A",
	"S",
	"D",
	"Z",
	"C",
	"4",
	"R",
	"F",
	"V",
}

const defaultCPUSpeed = 500

var screenColor color.RGBA

type machine struct {
	programPath string
	cpuSpeedHz  time.Duration
	video       [64 * 4 * 32 * 4 * 3]byte
}

func (m *machine) Load(memory []byte) {
	program, err := ioutil.ReadFile(m.programPath)
	if err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}
	copy(memory, program)
}

func (m *machine) Rand() *rand.Rand {
	return rand.New(rand.NewSource(time.Now().UnixNano()))
}

func (m *machine) BeginTone() {
	sdl.PauseAudio(false)
}

func (m *machine) EndTone() {
	sdl.PauseAudio(true)
}

func (m *machine) Key(code int) bool {
	scan := sdl.GetScancodeFromName(keymap[code])
	state := sdl.GetKeyboardState()
	return state[scan] != 0
}

func (m *machine) Draw(video []byte) {
	putPixel := func(x, y int, buffer []byte) {
		i := (y*64*4 + x) * 3
		buffer[i] = screenColor.B
		buffer[i+1] = screenColor.G
		buffer[i+2] = screenColor.R
	}

	destY := 0
	destX := 0

	for i := range m.video {
		m.video[i] = 0
	}

	for y := 0; y < 32; y++ {
		destY = y * 4
		for i := 0; i < 3; i++ {
			for x := 0; x < 64; x++ {
				destX = x * 4
				if video[y*64+x] > 0 {
					putPixel(destX, destY+i, m.video[:])
					putPixel(destX+1, destY+i, m.video[:])
					putPixel(destX+2, destY+i, m.video[:])
				}

			}
		}
	}
}

func updateTitle(window *sdl.Window, m *machine) {
	title := fmt.Sprintf("Chippy - %dHz - %s", m.cpuSpeedHz, path.Base(m.programPath))
	window.SetTitle(title)
}

func toggleFullscreen(window *sdl.Window) {
	isFullscreen := (window.GetFlags() & sdl.WINDOW_FULLSCREEN) != 0
	if isFullscreen {
		window.SetFullscreen(0)
		sdl.ShowCursor(1)
	} else {
		window.SetFullscreen(sdl.WINDOW_FULLSCREEN_DESKTOP)
		sdl.ShowCursor(0)
	}
}

func init() {
	flag.Parse()
	runtime.LockOSThread()
}

func main() {
	flags := flag.Args()
	if len(flags) != 1 {
		fmt.Println("Chippy - CHIP8 Emulator")
		fmt.Println("Copyright (C) 2016 Andreas T Jonsson")
		fmt.Printf("Version: %v\n\n", chip8.Version)
		fmt.Printf("usage: chippy [program]\n\n")
		return
	}

	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	screenColor.R = uint8(r.Intn(127)) + 128
	screenColor.G = uint8(r.Intn(127)) + 128
	screenColor.B = uint8(r.Intn(127)) + 128
	screenColor.A = 255

	sdl.Init(sdl.INIT_EVERYTHING)
	defer sdl.Quit()

	window, err := sdl.CreateWindow("", sdl.WINDOWPOS_UNDEFINED, sdl.WINDOWPOS_UNDEFINED, 640, 320, sdl.WINDOW_SHOWN)
	if err != nil {
		panic(err)
	}
	defer window.Destroy()

	renderer, err := sdl.CreateRenderer(window, -1, sdl.RENDERER_ACCELERATED)
	if err != nil {
		panic(err)
	}
	defer renderer.Destroy()

	sdl.SetHint(sdl.HINT_RENDER_SCALE_QUALITY, "linear")
	renderer.SetLogicalSize(64*4, 32*4)
	renderer.SetDrawColor(0, 0, 0, 255)

	texture, err := renderer.CreateTexture(sdl.PIXELFORMAT_BGR24, sdl.TEXTUREACCESS_STREAMING, 64*4, 32*4)
	if err != nil {
		panic(err)
	}
	defer texture.Destroy()

	var specIn sdl.AudioSpec
	specIn.Channels = 1
	specIn.Format = sdl.AUDIO_U8
	specIn.Freq = 11025
	specIn.Samples = 4096
	specIn.Callback = sdl.AudioCallback(C.audioCallback)

	if err := sdl.OpenAudio(&specIn, nil); err != nil {
		panic(err)
	}
	defer sdl.CloseAudio()

	m := machine{programPath: flags[0], cpuSpeedHz: defaultCPUSpeed}
	updateTitle(window, &m)

	sys := chip8.NewSystem(&m)

	tickRender := time.Tick(time.Second / 65)
	tickCPU := time.Tick(time.Second / m.cpuSpeedHz)

	for {
		select {
		case <-tickRender:
			for event := sdl.PollEvent(); event != nil; event = sdl.PollEvent() {
				switch t := event.(type) {
				case *sdl.QuitEvent:
					return
				case *sdl.KeyUpEvent:
					switch t.Keysym.Sym {
					case sdl.K_ESCAPE:
						return
					case sdl.K_BACKSPACE:
						sys.Reset()
					case sdl.K_g:
						toggleFullscreen(window)
					case sdl.K_p:
						if m.cpuSpeedHz < 2000 {
							m.cpuSpeedHz += 100
							updateTitle(window, &m)
							tickCPU = time.Tick(time.Second / m.cpuSpeedHz)
						}
					case sdl.K_m:
						if m.cpuSpeedHz > 100 {
							m.cpuSpeedHz -= 100
							updateTitle(window, &m)
							tickCPU = time.Tick(time.Second / m.cpuSpeedHz)
						}
					}
				}
			}

			sys.Refresh()

			renderer.Clear()
			texture.Update(nil, unsafe.Pointer(&m.video[0]), 64*4*3)
			renderer.Copy(texture, nil, nil)
			renderer.Present()
		case <-tickCPU:
			if err := sys.Step(); err != nil {
				panic(err)
			}
		}
	}
}

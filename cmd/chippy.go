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
#include <math.h>
void audioCallback(void *userdata, unsigned char *stream, int len) {
	for (int i = 0; i < len; i++) {
		stream[i] = (sin(i / 4) + 1) * 128;
	}
}
*/
import "C"

import (
	"flag"
	"fmt"
	"io/ioutil"
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

var (
	programPath string
	cpuSpeedHz  time.Duration = 500
)

type machine [64 * 32 * 3]byte

func (m *machine) Load(memory []byte) {
	program, err := ioutil.ReadFile(programPath)
	if err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}
	copy(memory, program)
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
	for i, x := range video {
		i *= 3
		c := x * 255

		m[i] = c
		m[i+1] = c
		m[i+2] = c
	}
}

func updateTitle(window *sdl.Window) {
	title := fmt.Sprintf("Chippy - %dHz - %s", cpuSpeedHz, path.Base(programPath))
	window.SetTitle(title)
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
	programPath = flags[0]

	sdl.Init(sdl.INIT_EVERYTHING)
	defer sdl.Quit()

	window, err := sdl.CreateWindow("", sdl.WINDOWPOS_UNDEFINED, sdl.WINDOWPOS_UNDEFINED, 640, 320, sdl.WINDOW_SHOWN)
	if err != nil {
		panic(err)
	}
	defer window.Destroy()
	updateTitle(window)

	renderer, err := sdl.CreateRenderer(window, -1, sdl.RENDERER_ACCELERATED)
	if err != nil {
		panic(err)
	}
	defer renderer.Destroy()

	sdl.SetHint(sdl.HINT_RENDER_SCALE_QUALITY, "0")
	renderer.SetLogicalSize(64, 32)
	renderer.SetDrawColor(0, 0, 0, 255)

	texture, err := renderer.CreateTexture(sdl.PIXELFORMAT_BGR24, sdl.TEXTUREACCESS_STREAMING, 64, 32)
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

	var m machine
	sys := chip8.NewSystem(&m)

	tickRender := time.Tick(time.Second / 65)
	tickCPU := time.Tick(time.Second / cpuSpeedHz)

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
					case sdl.K_p:
						if cpuSpeedHz < 2000 {
							cpuSpeedHz += 100
							updateTitle(window)
							tickCPU = time.Tick(time.Second / cpuSpeedHz)
						}
					case sdl.K_m:
						if cpuSpeedHz > 100 {
							cpuSpeedHz -= 100
							updateTitle(window)
							tickCPU = time.Tick(time.Second / cpuSpeedHz)
						}
					}
				}
			}

			sys.Refresh()

			renderer.Clear()
			texture.Update(nil, unsafe.Pointer(&m[0]), 64*3)
			renderer.Copy(texture, nil, nil)
			renderer.Present()
		case <-tickCPU:
			if err := sys.Step(); err != nil {
				panic(err)
			}
		}
	}
}

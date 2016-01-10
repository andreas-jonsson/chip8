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

import (
	"flag"
	"fmt"
	"io/ioutil"
	"runtime"
	"time"
	"unsafe"

	"github.com/andreas-jonsson/chip8/chip8"
	"github.com/veandco/go-sdl2/sdl"
)

var keymap = [16]string{
	"0",
	"1",
	"2",
	"3",
	"4",
	"5",
	"6",
	"7",
	"8",
	"9",
	"A",
	"B",
	"C",
	"D",
	"E",
	"F",
}

type machine [64 * 32 * 3]byte

func (m *machine) Load(memory []byte) {
	program, err := ioutil.ReadFile(flag.Args()[1])
	if err != nil {
		panic(err)
	}
	copy(memory, program)
}

func (m *machine) Beep() {
	fmt.Printf("\a")
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

func init() {
	runtime.LockOSThread()
}

func main() {
	if len(flag.Args()) != 2 {
		fmt.Println("Chippy - CHIP8 Emulator")
		fmt.Println("Copyright (C) 2016 Andreas T Jonsson\n")
		fmt.Println("usage: chippy [program]\n")
		return
	}

	sdl.Init(sdl.INIT_EVERYTHING)
	defer sdl.Quit()

	window, err := sdl.CreateWindow("Chippy", sdl.WINDOWPOS_UNDEFINED, sdl.WINDOWPOS_UNDEFINED, 640, 320, sdl.WINDOW_SHOWN)
	if err != nil {
		panic(err)
	}
	defer window.Destroy()

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

	var m machine
	sys := chip8.NewSystem(&m)

	tickRender := time.Tick(time.Second / 75)
	tickCPU := time.Tick(time.Second / 500)

	for {
		select {
		case <-tickRender:
			for event := sdl.PollEvent(); event != nil; event = sdl.PollEvent() {
				switch t := event.(type) {
				case *sdl.QuitEvent:
					return
				case *sdl.KeyUpEvent:
					if t.Keysym.Sym == sdl.K_ESCAPE {
						return
					}
				}
			}

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

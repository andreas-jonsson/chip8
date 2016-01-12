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
	"os"
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

var programPath string

type machine [64 * 32 * 3]byte

func (m *machine) Load(memory []byte) {
	program, err := ioutil.ReadFile(programPath)
	if err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}
	copy(memory, program)
}

func (m *machine) Beep() {
	//go func() {
	//	fmt.Printf("\a")
	//}()
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
	flag.Parse()
	runtime.LockOSThread()
}

func main() {
	flags := flag.Args()
	if len(flags) != 1 {
		fmt.Println("Chippy - CHIP8 Emulator")
		fmt.Println("Copyright (C) 2016 Andreas T Jonsson")
		fmt.Printf("Version: %v\n\n", chip8.Version)
		fmt.Println("usage: chippy [program]\n")
		return
	} else {
		programPath = flags[0]
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

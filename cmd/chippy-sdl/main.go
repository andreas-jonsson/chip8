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
	"log"
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
	video       [64 * 32]uint32
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
func (m *machine) SetCPUFrequency(freq int) {
	m.cpuSpeedHz = time.Duration(freq)
}

func (m *machine) Draw(video []byte) {
	pal := []uint32{
		0x00000000,
		0x00CC0033,
		0x00000099,
		0x00CC33CC,
		0x00006633,
		0x00666666,
		0x003333FF,
		0x006699FF,
		0x00996600,
		0x00FF6600,
		0x00999999,
		0x00FF9999,
		0x0000CC00,
		0x00FFFF00,
		0x0033FF99,
		0x00FFFFFF,
	}

	for offset, index := range video {
		m.video[offset] = pal[index]
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

func dumpSystem(sys *chip8.System, name string) {
	fmt.Println("writing system dump...")
	if fp, err := os.Create(fmt.Sprintf("%s.dump", name)); err == nil {
		sys.Dump(fp, name)
		fp.Close()
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

	window, err := sdl.CreateWindow("", sdl.WINDOWPOS_UNDEFINED, sdl.WINDOWPOS_UNDEFINED, 800, 600, sdl.WINDOW_SHOWN)
	if err != nil {
		log.Fatalln(err)
	}
	defer window.Destroy()

	renderer, err := sdl.CreateRenderer(window, -1, sdl.RENDERER_ACCELERATED)
	if err != nil {
		log.Fatalln(err)
	}
	defer renderer.Destroy()

	renderer.SetLogicalSize(800, 600)
	renderer.SetDrawColor(0, 0, 0, 255)

	texture, err := renderer.CreateTexture(sdl.PIXELFORMAT_ABGR8888, sdl.TEXTUREACCESS_STREAMING, 64, 32)
	if err != nil {
		log.Fatalln(err)
	}
	defer texture.Destroy()

	var specIn sdl.AudioSpec
	specIn.Channels = 1
	specIn.Format = sdl.AUDIO_U8
	specIn.Freq = 11025
	specIn.Samples = 4096
	specIn.Callback = sdl.AudioCallback(C.audioCallback)

	if err := sdl.OpenAudio(&specIn, nil); err != nil {
		log.Fatalln(err)
	}
	defer sdl.CloseAudio()

	m := machine{programPath: flags[0], cpuSpeedHz: defaultCPUSpeed}
	updateTitle(window, &m)

	sys := chip8.NewSystem(&m)

	cpuSpeedHz := m.cpuSpeedHz
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
					case sdl.K_g:
						toggleFullscreen(window)
					case sdl.K_p:
						if m.cpuSpeedHz < 2000 {
							m.cpuSpeedHz += 100
						}
					case sdl.K_m:
						if m.cpuSpeedHz > 100 {
							m.cpuSpeedHz -= 100
						}
					case sdl.K_d:
						if t.Keysym.Mod&sdl.KMOD_CTRL != 0 {
							dumpSystem(sys, flags[0])
						}
					}
				}
			}

			if cpuSpeedHz != m.cpuSpeedHz {
				cpuSpeedHz = m.cpuSpeedHz
				updateTitle(window, &m)
				tickCPU = time.Tick(time.Second / m.cpuSpeedHz)
			}

			sys.Refresh()

			renderer.Clear()
			texture.Update(nil, unsafe.Pointer(&m.video[0]), 64*4)
			renderer.Copy(texture, nil, nil)
			renderer.Present()
		case <-tickCPU:
			if err := sys.Step(); err != nil {
				dumpSystem(sys, flags[0])
				log.Fatalln(err)
			}
		}
	}
}

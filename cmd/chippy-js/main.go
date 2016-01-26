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

// +build js

package main

import (
	"fmt"
	"image/color"
	"math/rand"
	"strconv"
	"sync"
	"time"

	"github.com/andreas-jonsson/chip8/chip8"
	"github.com/gopherjs/gopherjs/js"
)

const (
	imgWidth        = 64 * 4
	imgHeight       = 32 * 4
	defaultCPUSpeed = 500
)

var (
	screenColor color.RGBA
	kbMapping   = [16]int{
		88,
		49,
		50,
		51,
		81,
		87,
		69,
		65,
		83,
		68,
		90,
		67,
		52,
		82,
		70,
		86,
	}
)

var kb struct {
	sync.Mutex
	keys map[int]bool
}

type machine struct {
	sync.Mutex

	program     []byte
	programName string
	cpuSpeedHz  time.Duration
	video       [64 * 4 * 32 * 4 * 3]byte
	canvas      *js.Object
}

func (m *machine) Load(memory []byte) {
	copy(memory, m.program)
}

func (m *machine) Rand() *rand.Rand {
	return rand.New(rand.NewSource(time.Now().UnixNano()))
}

func (m *machine) BeginTone() {
}

func (m *machine) EndTone() {
}

func (m *machine) Key(code int) bool {
	kb.Lock()
	defer kb.Unlock()
	return kb.keys[kbMapping[code]]
}

func (m *machine) Draw(video []byte) {
	m.Lock()
	defer m.Unlock()

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

func updateTitle(m *machine) {
	title := fmt.Sprintf("Chippy - %dHz - %s", m.cpuSpeedHz, m.programName)
	js.Global.Get("document").Set("title", title)
}

func openFile() (string, []byte) {
	document := js.Global.Get("document")
	inputElem := document.Call("createElement", "input")
	inputElem.Call("setAttribute", "type", "file")
	document.Get("body").Call("appendChild", inputElem)

	filec := make(chan *js.Object, 1)
	inputElem.Set("onchange", func(event *js.Object) {
		filec <- inputElem.Get("files").Index(0)
	})

	file := <-filec
	name := file.Get("name").String()
	reader := js.Global.Get("FileReader").New()

	bufc := make(chan []byte, 1)
	reader.Set("onloadend", func(event *js.Object) {
		bufc <- js.Global.Get("Uint8Array").New(reader.Get("result")).Interface().([]byte)
	})
	reader.Call("readAsArrayBuffer", file)
	data := <-bufc

	document.Get("body").Call("removeChild", inputElem)
	return name, data
}

func main() {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	screenColor.R = uint8(r.Intn(127)) + 128
	screenColor.G = uint8(r.Intn(127)) + 128
	screenColor.B = uint8(r.Intn(127)) + 128
	screenColor.A = 255

	kb.keys = make(map[int]bool)

	js.Global.Call("addEventListener", "load", func() { go start() })
}

func draw(m *machine, sys *chip8.System) {
	sys.Refresh()

	m.Lock()
	defer m.Unlock()

	ctx := m.canvas.Call("getContext", "2d")
	img := ctx.Call("getImageData", 0, 0, imgWidth, imgHeight)
	data := img.Get("data")

	arrBuf := js.Global.Get("ArrayBuffer").New(data.Length())
	buf8 := js.Global.Get("Uint8ClampedArray").New(arrBuf)
	buf32 := js.Global.Get("Uint32Array").New(arrBuf)

	buf := buf32.Interface().([]uint)

	for i := 0; i < len(m.video); i += 3 {
		buf[i/3] = 0xFF000000 | (uint(m.video[i]) << 16) | (uint(m.video[i+1]) << 8) | uint(m.video[i+2])
	}

	data.Call("set", buf8)
	ctx.Call("putImageData", img, 0, 0)

	js.Global.Call("requestAnimationFrame", func() { draw(m, sys) })
}

func start() {
	name, buffer := openFile()

	m := machine{programName: name, program: buffer, cpuSpeedHz: defaultCPUSpeed}
	updateTitle(&m)

	document := js.Global.Get("document")
	document.Set("onkeydown", func(e *js.Object) {
		kb.Lock()
		kb.keys[e.Get("keyCode").Int()] = true
		kb.Unlock()
	})

	document.Set("onkeyup", func(e *js.Object) {
		kb.Lock()
		kb.keys[e.Get("keyCode").Int()] = false
		kb.Unlock()
	})

	canvas := document.Call("createElement", "canvas")
	m.canvas = canvas

	canvas.Call("setAttribute", "width", strconv.Itoa(imgWidth))
	canvas.Call("setAttribute", "height", strconv.Itoa(imgHeight))
	canvas.Get("style").Set("width", strconv.Itoa(imgWidth*2)+"px")
	canvas.Get("style").Set("height", strconv.Itoa(imgHeight*2)+"px")
	document.Get("body").Call("appendChild", canvas)

	sys := chip8.NewSystem(&m)

	go func() {
		tickCPU := time.Tick(time.Second / m.cpuSpeedHz)
		for _ = range tickCPU {
			if err := sys.Step(); err != nil {
				panic(err)
			}
		}
	}()

	js.Global.Call("requestAnimationFrame", func() { draw(&m, sys) })
}

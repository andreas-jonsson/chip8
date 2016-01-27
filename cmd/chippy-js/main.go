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
	program     []byte
	programName string
	cpuSpeedHz  time.Duration
	video       [64 * 4 * 32 * 4 * 3]byte
	muteAudio   func(bool)
	canvas      *js.Object
}

func (m *machine) Load(memory []byte) {
	copy(memory, m.program)
}

func (m *machine) Rand() *rand.Rand {
	return rand.New(rand.NewSource(time.Now().UnixNano()))
}

func (m *machine) BeginTone() {
	m.muteAudio(false)
}

func (m *machine) EndTone() {
	m.muteAudio(true)
}

func (m *machine) Key(code int) bool {
	kb.Lock()
	defer kb.Unlock()
	return kb.keys[kbMapping[code]]
}

func putPixel(x, y int, buffer []byte, fill bool) {
	i := (y*64*4 + x) * 3
	if fill {
		buffer[i] = screenColor.B
		buffer[i+1] = screenColor.G
		buffer[i+2] = screenColor.R
	} else {
		buffer[i] = 0
		buffer[i+1] = 0
		buffer[i+2] = 0
	}
}

func (m *machine) Draw(video []byte) {
	destY, destX := 0, 0
	for y := 0; y < 32; y++ {
		destY = y * 4
		for i := 0; i < 3; i++ {
			for x := 0; x < 64; x++ {
				destX = x * 4
				if video[y*64+x] > 0 {
					for j := 0; j < 3; j++ {
						putPixel(destX+j, destY+i, m.video[:], true)
					}
				} else {
					for j := 0; j < 4; j++ {
						putPixel(destX+j, destY+i, m.video[:], false)
					}
				}
			}
		}
	}

	updateScreen(m.canvas, m.video[:])
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

func updateScreen(canvas *js.Object, video []byte) {
	ctx := canvas.Call("getContext", "2d")
	img := ctx.Call("getImageData", 0, 0, imgWidth, imgHeight)
	data := img.Get("data")

	arrBuf := js.Global.Get("ArrayBuffer").New(data.Length())
	buf8 := js.Global.Get("Uint8ClampedArray").New(arrBuf)
	buf32 := js.Global.Get("Uint32Array").New(arrBuf)

	buf := buf32.Interface().([]uint)

	for i := 0; i < len(video); i += 3 {
		buf[i/3] = 0xFF000000 | (uint(video[i]) << 16) | (uint(video[i+1]) << 8) | uint(video[i+2])
	}

	data.Call("set", buf8)
	ctx.Call("putImageData", img, 0, 0)
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

func start() {
	name, buffer := openFile()

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

	// Create canvas.
	canvas := document.Call("createElement", "canvas")

	canvas.Call("setAttribute", "width", strconv.Itoa(imgWidth))
	canvas.Call("setAttribute", "height", strconv.Itoa(imgHeight))
	canvas.Get("style").Set("width", strconv.Itoa(imgWidth*3)+"px")
	canvas.Get("style").Set("height", strconv.Itoa(imgHeight*3)+"px")
	document.Get("body").Call("appendChild", canvas)

	m := machine{programName: name, program: buffer, canvas: canvas, cpuSpeedHz: defaultCPUSpeed}
	updateTitle(&m)

	// Create audio.
	audioClass := js.Global.Get("AudioContext")
	if audioClass.String() == "undefined" {
		audioClass = js.Global.Get("webkitAudioContext")
	}

	if audioClass.String() != "undefined" {
		audioContext := audioClass.New()
		oscillator := audioContext.Call("createOscillator")
		gain := audioContext.Call("createGain")

		oscillator.Call("connect", gain)
		oscillator.Set("type", "square")
		oscillator.Get("frequency").Set("value", 500)
		oscillator.Call("start", "0")

		m.muteAudio = func(mute bool) {
			if audioClass != nil {
				dest := audioContext.Get("destination")
				if mute {
					gain.Call("disconnect", dest)
				} else {
					gain.Call("connect", dest)
				}
			}
		}
	} else {
		m.muteAudio = func(mute bool) {}
	}

	go func() {
		sys := chip8.NewSystem(&m)
		tickRender := time.Tick(time.Second / 32)
		tickCPU := time.Tick(time.Second / m.cpuSpeedHz)

		for {
			select {
			case <-tickRender:
				sys.Refresh()
			case <-tickCPU:
				if err := sys.Step(); err != nil {
					js.Global.Call("alert", err.Error())
				}
			}
		}
	}()
}

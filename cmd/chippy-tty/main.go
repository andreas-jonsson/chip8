// +build tty

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
	"math/rand"
	"os"
	"time"

	"github.com/andreas-jonsson/chip8/chip8"
	"github.com/nsf/termbox-go"
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

type machine struct {
	programPath string
	cpuSpeedHz  time.Duration
}

func (m *machine) Load(memory []byte) {
	program, err := ioutil.ReadFile(m.programPath)
	if err != nil {
		termbox.Close()
		fmt.Println(err)
		os.Exit(-1)
	}
	copy(memory, program)
}

func (m *machine) Rand() *rand.Rand {
	return rand.New(rand.NewSource(time.Now().UnixNano()))
}

func (m *machine) BeginTone() {}

func (m *machine) EndTone() {}

func (m *machine) Key(code int) bool {
	return false
}

func (m *machine) Draw(video []byte) {
	termbox.Clear(termbox.ColorDefault, termbox.ColorDefault)
	w, h := termbox.Size()

	for y := 0; y < 32 && y < h; y++ {
		for x := 0; x < 64 && x < w; x++ {
			if video[y*64+x] > 0 {
				termbox.SetCell(x, y, ' ', termbox.AttrReverse, termbox.AttrReverse)
			}
		}
	}

	termbox.Flush()
}

func init() {
	flag.Parse()
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

	if err := termbox.Init(); err != nil {
		panic(err)
	}
	defer termbox.Close()

	termbox.SetOutputMode(termbox.OutputNormal)
	termbox.Clear(termbox.ColorDefault, termbox.ColorDefault)
	termbox.Sync()

	m := machine{programPath: flags[0], cpuSpeedHz: defaultCPUSpeed}
	sys := chip8.NewSystem(&m)

	go func() {
		for _ = range time.Tick(time.Second / m.cpuSpeedHz) {
			termbox.Interrupt()
		}
	}()

	for {
		switch ev := termbox.PollEvent(); ev.Type {
		case termbox.EventKey:
			switch ev.Key {
			case termbox.KeyEsc:
				return
			}
		case termbox.EventInterrupt:
			sys.Refresh()
			if err := sys.Step(); err != nil {
				panic(err)
			}
		}
	}

}

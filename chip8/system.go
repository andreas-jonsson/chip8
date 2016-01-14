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

package chip8

import (
	"fmt"
	"math/rand"
	"time"
)

const Version = "1.1.0"

var fontset = [80]byte{
	0xF0, 0x90, 0x90, 0x90, 0xF0, // 0
	0x20, 0x60, 0x20, 0x20, 0x70, // 1
	0xF0, 0x10, 0xF0, 0x80, 0xF0, // 2
	0xF0, 0x10, 0xF0, 0x10, 0xF0, // 3
	0x90, 0x90, 0xF0, 0x10, 0x10, // 4
	0xF0, 0x80, 0xF0, 0x10, 0xF0, // 5
	0xF0, 0x80, 0xF0, 0x90, 0xF0, // 6
	0xF0, 0x10, 0x20, 0x40, 0x40, // 7
	0xF0, 0x90, 0xF0, 0x90, 0xF0, // 8
	0xF0, 0x90, 0xF0, 0x10, 0xF0, // 9
	0xF0, 0x90, 0xF0, 0x90, 0x90, // A
	0xE0, 0x90, 0xE0, 0x90, 0xE0, // B
	0xF0, 0x80, 0x80, 0x80, 0xF0, // C
	0xE0, 0x90, 0x90, 0x90, 0xE0, // D
	0xF0, 0x80, 0xF0, 0x80, 0xF0, // E
	0xF0, 0x80, 0xF0, 0x80, 0x80, // F
}

type InputOutput interface {
	Load(memory []byte)
	Draw(video []byte)
	Key(code int) bool
	BeginTone()
	EndTone()
}

type System struct {
	pc, sp, i uint16
	v         [16]byte

	stack  [16]uint16
	memory [4096]byte
	video  [2048]byte

	delayTimer, soundTimer byte
	io                     InputOutput

	lastTick time.Time
	draw     bool
}

func (sys *System) Reset() {
	sys.pc = 0x200
	sys.sp = 0x0
	sys.i = 0x0

	sys.delayTimer = 0
	sys.soundTimer = 0

	sys.lastTick = time.Now()

	for i := range sys.v {
		sys.v[i] = 0x0
	}

	for i := range sys.memory {
		sys.memory[i] = 0
	}

	for i, x := range fontset {
		sys.memory[i] = x
	}

	sys.clearScreen()
	sys.io.Load(sys.memory[512:])
}

func (sys *System) clearScreen() {
	for i := range sys.video {
		sys.video[i] = 0
	}
	sys.draw = true
}

func (sys *System) tickTimers() {
	if time.Since(sys.lastTick) < time.Second/60 {
		return
	}
	sys.lastTick = time.Now()

	if sys.delayTimer > 0 {
		sys.delayTimer--
	}

	if sys.soundTimer > 0 {
		sys.soundTimer--
		if sys.soundTimer == 0 {
			sys.io.EndTone()
		}
	}
}

func (sys *System) op0(opcode uint16) error {
	switch opcode & 0xF {
	case 0x0:
		sys.clearScreen()
		sys.pc += 2
	case 0xE:
		sys.sp--
		sys.pc = sys.stack[sys.sp&0xF] + 2
	default:
		if err := fmt.Errorf("invalid opcode: %v", opcode&0xF); err != nil {
			return err
		}
	}
	return nil
}

func (sys *System) op8(opcode uint16) error {
	switch opcode & 0xF {
	case 0x0:
		sys.v[(opcode&0xF00)>>8] = sys.v[(opcode&0xF0)>>4]
		sys.pc += 2
	case 0x1:
		sys.v[(opcode&0xF00)>>8] = sys.v[(opcode&0xF00)>>8] | sys.v[(opcode&0xF0)>>4]
		sys.pc += 2
	case 0x2:
		sys.v[(opcode&0xF00)>>8] = sys.v[(opcode&0xF00)>>8] & sys.v[(opcode&0xF0)>>4]
		sys.pc += 2
	case 0x3:
		sys.v[(opcode&0xF00)>>8] = sys.v[(opcode&0xF00)>>8] ^ sys.v[(opcode&0xF0)>>4]
		sys.pc += 2
	case 0x4:
		res := int(sys.v[(opcode&0xF00)>>8]) + int(sys.v[(opcode&0xF0)>>4])
		if res < 256 {
			sys.v[0xF] &= 0
		} else {
			sys.v[0xF] = 1
		}

		sys.v[(opcode&0xF00)>>8] = byte(res)
		sys.pc += 2
	case 0x5:
		res := int(sys.v[(opcode&0xF00)>>8]) - int(sys.v[(opcode&0xF0)>>4])
		if res >= 0 {
			sys.v[0xF] = 1
		} else {
			sys.v[0xF] &= 0
		}

		sys.v[(opcode&0xF00)>>8] = byte(res)
		sys.pc += 2
	case 0x6:
		sys.v[0xF] = sys.v[(opcode&0xF00)>>8] & 7
		sys.v[(opcode&0xF00)>>8] = sys.v[(opcode&0xF00)>>8] >> 1
		sys.pc += 2
	case 0x7:
		res := int(sys.v[(opcode&0xF00)>>8]) - int(sys.v[(opcode&0xF0)>>4])
		if res > 0 {
			sys.v[0xF] = 1
		} else {
			sys.v[0xF] &= 0
		}

		sys.v[(opcode&0xF00)>>8] = byte(res)
		sys.pc += 2
	case 0xE:
		sys.v[0xF] = sys.v[(opcode&0xF00)>>8] >> 7
		sys.v[(opcode&0xF00)>>8] = sys.v[(opcode&0xF00)>>8] << 1
		sys.pc += 2
	default:
		if err := fmt.Errorf("invalid opcode: %v", opcode&0xF); err != nil {
			return err
		}
	}
	return nil
}

func (sys *System) opE(opcode uint16) error {
	switch opcode & 0xF {
	case 0x1:
		if !sys.io.Key(int(sys.v[(opcode&0xF00)>>8])) {
			sys.pc += 4
		} else {
			sys.pc += 2
		}
	case 0xE:
		if sys.io.Key(int(sys.v[(opcode&0xF00)>>8])) {
			sys.pc += 4
		} else {
			sys.pc += 2
		}
	default:
		if err := fmt.Errorf("invalid opcode: %v", opcode&0xF); err != nil {
			return err
		}
	}
	return nil
}

func (sys *System) opF(opcode uint16) error {
	switch opcode & 0xFF {
	case 0x7:
		sys.v[(opcode&0xF00)>>8] = sys.delayTimer
		sys.pc += 2
	case 0xA:
		for i := 0; i < 16; i++ {
			if sys.io.Key(i) {
				sys.v[(opcode&0xF00)>>8] = byte(i)
				sys.pc += 2
			}
		}
	case 0x15:
		sys.delayTimer = sys.v[(opcode&0xF00)>>8]
		sys.pc += 2
	case 0x18:
		t := sys.v[(opcode&0xF00)>>8]
		if sys.delayTimer == 0 && t > 0 {
			sys.io.BeginTone()
		}
		sys.soundTimer = t
		sys.pc += 2
	case 0x1E:
		sys.i += uint16(sys.v[(opcode&0xF00)>>8])
		sys.pc += 2
	case 0x29:
		sys.i = uint16(sys.v[(opcode&0xF00)>>8]) * 5
		sys.pc += 2
	case 0x33:
		sys.memory[sys.i] = sys.v[(opcode&0xF00)>>8] / 100
		sys.memory[sys.i+1] = (sys.v[(opcode&0xF00)>>8] / 10) % 10
		sys.memory[sys.i+2] = sys.v[(opcode&0xF00)>>8] % 10
		sys.pc += 2
	case 0x55:
		for i := uint16(0); i <= ((opcode & 0xF00) >> 8); i++ {
			sys.memory[sys.i+i] = sys.v[i]
		}
		sys.pc += 2
	case 0x65:
		for i := uint16(0); i <= ((opcode & 0xF00) >> 8); i++ {
			sys.v[i] = sys.memory[sys.i+i]
		}
		sys.pc += 2
	default:
		if err := fmt.Errorf("invalid opcode: %v", opcode&0xFF); err != nil {
			return err
		}
	}

	return nil
}

func (sys *System) Step() error {
	opcode := uint16(sys.memory[sys.pc])<<8 | uint16(sys.memory[sys.pc+1])

	switch opcode & 0xF000 {
	case 0x0:
		if err := sys.op0(opcode); err != nil {
			return err
		}
	case 0x1000:
		sys.pc = opcode & 0xFFF
	case 0x2000:
		sys.stack[sys.sp] = sys.pc
		sys.sp++
		sys.pc = opcode & 0xFFF
	case 0x3000:
		if sys.v[(opcode&0xF00)>>8] == byte(opcode&0xFF) {
			sys.pc += 4
		} else {
			sys.pc += 2
		}
	case 0x4000:
		if sys.v[(opcode&0xF00)>>8] != byte(opcode&0xFF) {
			sys.pc += 4
		} else {
			sys.pc += 2
		}
	case 0x5000:
		if sys.v[(opcode&0xF00)>>8] == sys.v[(opcode&0xF0)>>4] {
			sys.pc += 4
		} else {
			sys.pc += 2
		}
	case 0x6000:
		sys.v[(opcode&0xF00)>>8] = byte(opcode & 0xFF)
		sys.pc += 2
	case 0x7000:
		sys.v[(opcode&0xF00)>>8] += byte(opcode & 0xFF)
		sys.pc += 2
	case 0x8000:
		sys.op8(opcode)
	case 0x9000:
		if sys.v[(opcode&0xF00)>>8] != sys.v[(opcode&0xF0)>>4] {
			sys.pc += 4
		} else {
			sys.pc += 2
		}
	case 0xA000:
		sys.i = opcode & 0xFFF
		sys.pc += 2
	case 0xB000:
		sys.pc = (opcode & 0xFFF) + uint16(sys.v[0])
	case 0xC000:
		sys.v[(opcode&0xF00)>>8] = byte(rand.Intn(255)) & byte(opcode&0xFF)
		sys.pc += 2
	case 0xD000:
		x := uint16(sys.v[(opcode&0xF00)>>8])
		y := uint16(sys.v[(opcode&0xF0)>>4])
		height := opcode & 0xF

		sys.v[0xF] = 0
		for yline := uint16(0); yline < height; yline++ {
			pixel := sys.memory[sys.i+yline]
			for xline := uint16(0); xline < 8; xline++ {
				if (pixel & (0x80 >> xline)) != 0 {
					offset := x + xline + ((y + yline) * 64)
					if len(sys.video) > int(offset) && sys.video[offset] != 0 {
						sys.v[0xF] = 1
					}

					offset = x + xline + ((y + yline) * 64)
					if len(sys.video) > int(offset) {
						sys.video[offset] ^= 1
					}
				}
			}
		}

		sys.pc += 2
		sys.draw = true
	case 0xE000:
		if err := sys.opE(opcode); err != nil {
			return err
		}
	case 0xF000:
		if err := sys.opF(opcode); err != nil {
			return err
		}
	default:
		if err := fmt.Errorf("invalid opcode: %v", opcode&0xF000); err != nil {
			return err
		}
	}

	sys.tickTimers()
	return nil
}

func (sys *System) Refresh() {
	if sys.draw {
		sys.draw = false
		sys.io.Draw(sys.video[:])
	}
}

func NewSystem(io InputOutput) *System {
	sys := new(System)
	sys.io = io
	sys.Reset()
	return sys
}

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
	"bufio"
	"encoding/binary"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
	"strings"
)

const version = "0.1.0"

type (
	patchInfo struct {
		inst uint16
		lable,
		file string
		line int
	}

	assembler struct {
		file   string
		line   int
		offset uint16

		writer  io.WriteSeeker
		patches map[uint16]patchInfo
		lables  map[string]uint16
	}
)

func (asm *assembler) syntaxError() {
	log.Fatalf("syntax error, %s : %d\n", asm.file, asm.line)
}

func (asm *assembler) checkLen(args []string, n int) {
	if len(args) != n {
		asm.syntaxError()
	}
}

func (asm *assembler) parseNumber(s string) (uint16, error) {
	var (
		n   uint64
		err error
	)

	if strings.HasPrefix(s, "$") {
		n, err = strconv.ParseUint(s[1:], 16, 16)
		if err == nil {
			return uint16(n), nil
		}
	} else if strings.HasPrefix(s, "%") {
		n, err = strconv.ParseUint(s[1:], 2, 16)
		if err == nil {
			return uint16(n), nil
		}
	} else {
		n, err = strconv.ParseUint(s, 10, 16)
		if err == nil {
			return uint16(n), nil
		}
	}

	return 0, err
}

func (asm *assembler) parseRegName(s string) uint16 {
	if s[0] == 'v' {
		n, err := strconv.ParseUint(s[1:], 16, 4)
		if err == nil {
			return uint16(n)
		}
	}

	asm.syntaxError()
	return 0
}

func (asm *assembler) writeUint8(value byte) {
	if err := binary.Write(asm.writer, binary.BigEndian, value); err != nil {
		log.Fatalln(err)
	}
}

func (asm *assembler) writeUint16(value uint16) {
	if err := binary.Write(asm.writer, binary.BigEndian, value); err != nil {
		log.Fatalln(err)
	}
}

func (asm *assembler) writeOpcode(args []string) uint16 {
	switch args[0] {
	case ".":
		n, err := asm.parseNumber(args[1])
		if err != nil {
			asm.syntaxError()
		}

		asm.writeUint8(byte(n))
		nof := uint16(len(args) - 1)
		asm.offset += nof
		return nof
	case "..":
		n, err := asm.parseNumber(args[1])
		if err != nil {
			asm.syntaxError()
		}

		asm.writeUint16(n)
		nof := uint16(len(args)-1) * 2
		asm.offset += nof
		return nof
	case "scr":
		asm.checkLen(args, 2)
		n, err := asm.parseNumber(args[1])
		if err != nil {
			asm.syntaxError()
		}
		asm.writeUint16(0xC0 | (n & 0xF))
	case "clr":
		asm.checkLen(args, 1)
		asm.writeUint16(0xE0)
	case "rts":
		asm.checkLen(args, 1)
		asm.writeUint16(0xEE)
	case "scrr":
		asm.checkLen(args, 1)
		asm.writeUint16(0xFB)
	case "scrl":
		asm.checkLen(args, 1)
		asm.writeUint16(0xFC)
	case "halt":
		asm.checkLen(args, 1)
		asm.writeUint16(0xFD)
	case "low":
		asm.checkLen(args, 1)
		asm.writeUint16(0xFE)
	case "high":
		asm.checkLen(args, 1)
		asm.writeUint16(0xFF)
	case "jump", "call", "loadi", "jump0", "sys":
		asm.checkLen(args, 2)

		var inst uint16
		switch args[0] {
		case "jump":
			inst = 0x1000
		case "call":
			inst = 0x2000
		case "loadi":
			inst = 0xA000
		case "jump0":
			inst = 0xB000
		case "sys":
			inst = 0x0000
		default:
			panic(nil)
		}

		lable := args[1]
		n, err := asm.parseNumber(lable)
		if err != nil {
			asm.patches[asm.offset] = patchInfo{inst, lable, asm.file, asm.line}
		}

		asm.writeUint16(inst | (n & 0x0FFF))
	case "ske", "skne", "load", "add", "rand":
		asm.checkLen(args, 3)

		reg := asm.parseRegName(args[1])
		n, err := asm.parseNumber(args[2])
		if err != nil {
			asm.syntaxError()
		}

		var inst uint16
		switch args[0] {
		case "ske":
			inst = 0x3000
		case "skne":
			inst = 0x4000
		case "load":
			inst = 0x6000
		case "add":
			inst = 0x7000
		case "rand":
			inst = 0xC000
		default:
			panic(nil)
		}

		asm.writeUint16(inst | (reg << 8) | (n & 0x00FF))
	case "skre", "move", "or", "and", "xor", "addr", "sub", "subr", "sknre":
		asm.checkLen(args, 3)

		reg0 := asm.parseRegName(args[1])
		reg1 := asm.parseRegName(args[2])

		var inst uint16
		switch args[0] {
		case "skre":
			inst = 0x5000
		case "move":
			inst = 0x8000
		case "or":
			inst = 0x8001
		case "and":
			inst = 0x8002
		case "xor":
			inst = 0x8003
		case "addr":
			inst = 0x8004
		case "sub":
			inst = 0x8005
		case "subr":
			inst = 0x8007
		case "sknre":
			inst = 0x9000
		default:
			panic(nil)
		}

		asm.writeUint16(inst | (reg0 << 8) | (reg1 << 4))
	case "shr", "shl", "skp", "sknp", "moved", "keyd", "loadd", "loads", "addi", "ldspr", "bcd", "stor", "read":
		asm.checkLen(args, 2)

		reg := asm.parseRegName(args[1])

		var inst uint16
		switch args[0] {
		case "shr":
			inst = 0x8006
		case "shl":
			inst = 0x800E
		case "skp":
			inst = 0xE09E
		case "sknp":
			inst = 0xE0A1
		case "moved":
			inst = 0xF007
		case "keyd":
			inst = 0xF00A
		case "loadd":
			inst = 0xF015
		case "loads":
			inst = 0xF018
		case "addi":
			inst = 0xF01E
		case "ldspr":
			inst = 0xF029
		case "bcd":
			inst = 0xF033
		case "stor":
			inst = 0xF055
		case "read":
			inst = 0xF065
		}

		asm.writeUint16(inst | (reg << 8))
	case "draw":
		asm.checkLen(args, 4)

		reg0 := asm.parseRegName(args[1])
		reg1 := asm.parseRegName(args[2])

		n, err := asm.parseNumber(args[3])
		if err != nil {
			asm.syntaxError()
		}

		asm.writeUint16(0xD000 | (reg0 << 8) | (reg1 << 4) | (n & 0x000F))
	default:
		asm.syntaxError()
	}

	asm.offset += 2
	return 2
}

func (asm *assembler) patchProgram() {
	for offset, info := range asm.patches {
		addr, ok := asm.lables[info.lable]
		if !ok {
			log.Fatalf("unknown lable '%s', %s : %d\n", info.lable, info.file, info.line)
		}

		asm.writer.Seek(int64(offset), 0)
		asm.writeUint16(info.inst | ((addr + 0x200) & 0x0FFF))
	}
	asm.writer.Seek(0, 2)
}

func (asm *assembler) saveLable(lable string) {
	asm.lables[lable] = asm.offset
}

func main() {
	fmt.Println("CHIP8 Assembler")
	fmt.Println("Copyright (C) 2016 Andreas T Jonsson")
	fmt.Printf("Version: %v\n\n", version)

	flag.Parse()
	flags := flag.Args()
	if len(flags) != 2 {
		fmt.Println("usage: prog <input.asm> <output.ch8>")
		return
	}

	fileName := flags[0]
	fp, err := os.Open(fileName)
	if err != nil {
		log.Fatalln(err)
	}
	defer fp.Close()

	outFile := flags[1]
	ofp, err := os.Create(outFile)
	if err != nil {
		log.Fatalln(err)
	}
	defer ofp.Close()

	mfp, err := os.Create(outFile + ".debug")
	if err != nil {
		log.Fatalln(err)
	}
	defer mfp.Close()
	fmt.Fprintln(mfp, fileName)

	asm := &assembler{
		file:    fileName,
		offset:  0,
		lables:  make(map[string]uint16),
		patches: make(map[uint16]patchInfo),
		writer:  ofp,
	}

	scanner := bufio.NewScanner(fp)

	for asm.line = 1; scanner.Scan(); asm.line++ {
		line := scanner.Text()

		for i, c := range line {
			if c == ';' {
				line = strings.TrimSpace(line[:i])
				break
			}
		}

		args := strings.Fields(line)
		argsLen := len(args)

		if argsLen == 0 {
			continue
		}

		first := args[0]
		l := len(first)

		if argsLen == 1 && first[l-1:] == ":" {
			lable := first[:l-1]
			asm.saveLable(lable)
			continue
		}

		for n := asm.writeOpcode(args); n > 0; n-- {
			fmt.Fprintln(mfp, asm.line)
		}
	}

	if err = scanner.Err(); err != nil {
		asm.syntaxError()
	}

	asm.patchProgram()
	size, _ := ofp.Seek(0, 2)
	fmt.Printf("program size: %d bytes\n", size)
}

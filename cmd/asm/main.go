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
	"io"
	"log"
	"os"
	"strconv"
	"strings"
)

func syntaxError(file string, ln int) {
	log.Fatalf("syntax error, %s : %d\n", file, ln)
}

func checkLen(args []string, n int, file string, ln int) {
	if len(args) != n {
		syntaxError(file, ln)
	}
}

func parseNumber(s string) (uint16, error) {
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

func parseRegName(s, file string, ln int) uint16 {
	if s[0] == 'v' {
		n, err := strconv.ParseUint(s[1:], 10, 4)
		if err == nil {
			return uint16(n)
		}
	}

	syntaxError(file, ln)
	return 0
}

func writeUint8(writer io.Writer, value byte) {
	if err := binary.Write(writer, binary.BigEndian, value); err != nil {
		log.Fatalln(err)
	}
}

func writeUint16(writer io.Writer, value uint16) {
	if err := binary.Write(writer, binary.BigEndian, value); err != nil {
		log.Fatalln(err)
	}
}

func writeOpcode(writer io.Writer, args []string, lables map[string]uint16, file string, ln int) uint16 {
	switch args[0] {
	case ".":
		n, err := parseNumber(args[1])
		if err != nil {
			syntaxError(file, ln)
		}

		writeUint8(writer, byte(n))
		return uint16(len(args)-1) * 2
	case "..":
		n, err := parseNumber(args[1])
		if err != nil {
			syntaxError(file, ln)
		}

		writeUint16(writer, n)
		return uint16(len(args)-1) * 2
	case "clr":
		checkLen(args, 1, file, ln)
		writeUint16(writer, 0xE0)
	case "rts":
		checkLen(args, 1, file, ln)
		writeUint16(writer, 0xEE)
	case "jump", "call", "loadi", "jumpi":
		checkLen(args, 2, file, ln)

		lable := args[1]
		n, err := parseNumber(lable)
		if err != nil {
			addr, ok := lables[lable]
			if !ok {
				log.Fatalf("unknown lable '%s', %s : %d\n", lable, file, ln)
			}
			n = addr + 0x200
		}

		var inst uint16
		switch args[0] {
		case "jump":
			inst = 0x1000
		case "call":
			inst = 0x2000
		case "loadi":
			inst = 0xA000
		case "jumpi":
			inst = 0xB000
		default:
			panic(nil)
		}

		writeUint16(writer, inst|(n&0x0FFF))
	case "ske", "skne", "load", "add", "rand":
		checkLen(args, 3, file, ln)

		reg := parseRegName(args[1], file, ln)
		n, err := parseNumber(args[2])
		if err != nil {
			syntaxError(file, ln)
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

		writeUint16(writer, inst|(reg<<8)|(n&0x00FF))
	case "skre", "move", "or", "and", "xor", "addr", "sub", "subr", "sknre":
		checkLen(args, 3, file, ln)

		reg0 := parseRegName(args[1], file, ln)
		reg1 := parseRegName(args[2], file, ln)

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

		writeUint16(writer, inst|(reg0<<8)|(reg1<<4))
	case "shr", "shl", "skp", "sknp", "moved", "keyd", "loadd", "loads", "addi", "ldspr", "bcd", "stor", "read":
		checkLen(args, 2, file, ln)

		reg := parseRegName(args[1], file, ln)

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

		writeUint16(writer, inst|(reg<<8))
	case "draw":
		checkLen(args, 4, file, ln)

		reg0 := parseRegName(args[1], file, ln)
		reg1 := parseRegName(args[2], file, ln)

		n, err := parseNumber(args[3])
		if err != nil {
			syntaxError(file, ln)
		}

		writeUint16(writer, 0xD000|(reg0<<8)|(reg1<<4)|(n&0x000F))
	default:
		syntaxError(file, ln)
	}

	return 2
}

func main() {
	fileName := "test.asm"
	fp, err := os.Open(fileName)
	if err != nil {
		log.Fatalln(err)
	}
	defer fp.Close()

	offset := uint16(0)
	lables := make(map[string]uint16)
	scanner := bufio.NewScanner(fp)

	var ln int
	for ln = 1; scanner.Scan(); ln++ {
		line := scanner.Text()

		for i, c := range line {
			if c == ';' {
				line = strings.TrimSpace(line[:i])
				break
			}
		}

		var args []string
		split := strings.SplitN(line, " ", 3)

		for _, s := range split {
			s := strings.TrimSpace(s)
			if len(s) > 0 {
				args = append(args, s)
			}
		}

		argsLen := len(args)
		if argsLen == 0 {
			continue
		}

		first := args[0]
		l := len(first)

		if argsLen == 1 && first[l-1:] == ":" {
			lable := first[:l-1]
			lables[lable] = offset
			continue
		}

		offset += writeOpcode(os.Stdout, args, lables, fileName, ln)
	}

	if err := scanner.Err(); err != nil {
		syntaxError(fileName, ln)
	}
}

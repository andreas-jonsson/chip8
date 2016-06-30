# Chip8 Color - Assembler

## Mnemonic Table

| Mnemonic | Opcode | Operands | Description |
| -------- | ------ | :------: | ----------- |
| `clr`    | `00E0` | 0 | Clear the screen                                               |
| `rts`    | `00EE` | 0 | Return from subroutine                                         |
| `jump`   | `1nnn` | 1 | Jump to address `nnn`                                          |
| `call`   | `2nnn` | 1 | Call routine at address `nnn`                                  |
| `ske`    | `3snn` | 2 | Skip next instruction if register `s` equals `nn`              |
| `skne`   | `4snn` | 2 | Do not skip next instruction if register `s` equals `nn`       |
| `skre`   | `5st0` | 2 | Skip if register `s` equals register `t`                       |
| `load`   | `6snn` | 2 | Load register `s` with value `nn`                              |
| `add`    | `7snn` | 2 | Add value `nn` to register `s`                                 |
| `move`   | `8st0` | 2 | Move value from register `s` to register `t`                   |
| `or`     | `8st1` | 2 | Perform logical OR on register `s` and `t` and store in `t`    |
| `and`    | `8st2` | 2 | Perform logical AND on register `s` and `t` and store in `t`   |
| `xor`    | `8st3` | 2 | Perform logical XOR on register `s` and `t` and store in `t`   |
| `addr`   | `8st4` | 2 | Add `s` to `t` and store in `s` - register `F` set on carry    |
| `sub`    | `8st5` | 2 | Subtract `s` from `t` and store in `s` - register `F` set on !borrow         |
| `shr`    | `8s06` | 1 | Shift bits in register `s` 1 bit to the right - bit 0 shifts to register `F` |
| `subr`   | `8st7` | 2 | Subtract `t` from `s` and store in `s` - register `F` set on !borrow         |
| `shl`    | `8s0E` | 1 | Shift bits in register `s` 1 bit to the left - bit 7 shifts to register `F`  |
| `skrne`  | `9st0` | 2 | Skip next instruction if register `s` not equal register `t`   |
| `loadi`  | `Annn` | 1 | Load index with value `nnn`                                    |
| `jump0`  | `Bnnn` | 1 | Jump to address `nnn` + v0                                  |
| `rand`   | `Ctnn` | 2 | Generate random number between 0 and `nn` and store in `t`     |
| `draw`   | `Dstn` | 3 | Draw `n` byte sprite at x location reg `s`, y location reg `t` |
| `skp`    | `Es9E` | 1 | Skip the following instruction if the key value stored in register `s` is pressed |
| `sknp`   | `EsA1` | 1 | Skip the following instruction if the key value stored in register `s` is not pressed |
| `moved`  | `Ft07` | 1 | Move delay timer value into register `t`                       |
| `keyd`   | `Ft0A` | 1 | Wait for keypress and store in register `t`                    |
| `loadd`  | `Fs15` | 1 | Load delay timer with value in register `s`                    |
| `loads`  | `Fs18` | 1 | Load sound timer with value in register `s`                    |
| `addi`   | `Fs1E` | 1 | Add value in register `s` to index                             |
| `ldspr`  | `Fs29` | 1 | Load index with sprite from register `s`                       |
| `bcd`    | `Fs33` | 1 | Store the binary coded decimal value of register `s` at index  |
| `stor`   | `Fs55` | 1 | Store the values of register `s` registers at index            |
| `read`   | `Fs65` | 1 | Read back the stored values at index into registers            |

#### Color extension

| Mnemonic | Opcode | Operands | Description |
| -------- | ------ | :------: | ----------- |
| `frq`    | `0nC0` | 1 | Set CPU frequency `n` hundred hz                     |
| `reset`  | `00FD` | 0 | System reset                                         |
| `swp`    | `00FE` | 0 | Swap back-buffer and enables color mode              |
| `fgc`    | `Fn75` | 1 | Set sprite color                                     |
| `bgc`    | `Fn85` | 1 | Set clear color                                      |

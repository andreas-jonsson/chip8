; Chip8 Color version of PONG

; V0-V3  are scratch registers
; V4     X coord. of score
; V5     Y coord. of score
; V6     X coord. of ball
; V7     Y coord. of ball
; V8     X direction of ball motion
; V9     Y direction of ball motion
; VA     X coord. of left player paddle
; VB     Y coord. of left player paddle
; VC     X coord. of right player paddle
; VD     Y coord. of right player paddle
; VE     Score
; VF     collision detection


; start
    load    v0 50
    sys     $100        ; Set CPU frequency to V0 * 10 (500hz)

    load    v0 40
    sys     $103        ; Set background color (keep sprite color white)

    load	va 2		; Set left player X coord
    load	vb 12		; Set left player Y coord
    load	vc 63		; Set right player X coord
    load	vd 12		; Set right player Y coord

    loadi	Paddle		; Get address of paddle sprite
    draw	va vb 6		; Draw left paddle
    draw	vc vd 6		; Draw right paddle

    load	ve 0		; Set score to 00
    call	Draw_Score	; Draw score

    load	v6 3		; Set X coord. of ball to 3
    load	v8 2		; Set ball X direction to right


Big_Loop:
    load  	v0 $60 		; Set V0=delay before ball launch
    loadd 	v0  		; Set delay timer to V0


DT_Loop:
    moved	v0			; Read delay timer into V0
    ske  	v0 0		; Skip next instruction if V0=0
	jump	DT_Loop		; Read again delay timer if not 0

    rand 	v7 23		; Set Y coord. to rand # AND 23 (0...23)
    add 	v7 8		; And adjust it to is 8...31

    load  	v9 $ff		; Set ball Y direction to up
    loadi 	Ball		; Get address of ball sprite
    draw 	v6 v7 1		; Draw ball


Padl_Loop:
    loadi   Paddle      ; Get address of paddle sprite
    draw    va vb 6     ; Draw left paddle
    draw    vc vd 6     ; Draw right paddle

    load    v0 1        ; Set V0 to KEY 1
    sknp    v0          ; Skip next instruction if KEY in 1 is not pressed
    add     vb $fe      ; Subtract 2 from Y coord. of left paddle

    load    v0 4        ; Set V0 to KEY 4
    sknp    v0          ; Skip next instruction if KEY in 4 is not pressed
    add     vb 2        ; Add 2 to Y coord. of left paddle

    load    v0 31       ; Set V0 to max Y coord.  | These three lines are here to
    and     vb v0       ; AND VB with V0          | adjust the paddle position if
    draw    va vb 6     ; Draw left paddle        | it is out of the screen

    load    v0 $0c      ; Set V0 to KEY C
    sknp    v0          ; Skip next instruction if KEY in C is not pressed
    add     vd $fe      ; Subtract 2 from Y coord. of right paddle

    load    v0 $0d      ; Set V0 to KEY D
    sknp    v0          ; Skip next instruction if KEY in D is not pressed
    add     vd 2        ; Add 2 to Y coord. of right paddle

    load    v0 31       ; Set V0 to max Y coord.  | These three lines are here to
    and     vd v0       ; AND VD with V0          | adjust the paddle position if
    draw    vc vd 6     ; Draw right paddle       | it is out of the screen

    loadi   Ball        ; Get address of ball sprite
    draw    v6 v7 1     ; Draw ball

    addr    v6 v8       ; Compute next X coord of the ball
    addr    v7 v9       ; Compute next Y coord of the ball

    load    v0 63       ; Set V0 to max X location
    and     v6 v0       ; AND V6 with V0

    load    v1 31       ; Set V1 to max Y location
    and     v7 v1       ; AND V7 with V1

    skne    v6 2        ; Skip next instruction if ball not at left
    jump    Left_Side

    skne    v6 63       ; Skip next instruction if ball not at right
    jump    Right_Side


Ball_Loop:
    skne    v7 31       ; Skip next instruction if ball not at bottom
    load    v9 $ff      ; Set Y direction to up

    skne    v7 0        ; Skip next instruction if ball not at top
    load    v9 1        ; Set Y direction to down

    draw    v6 v7 1     ; Draw ball
    jump    Padl_Loop


Left_Side:
    load    v8 2        ; Set X direction to right
    load    v3 1        ; Set V3 to 1 in case left player misses ball
    move    v0 v7       ; Set V0 to V7 Y coord. of ball
    sub     v0 vb       ; Subtract position of paddle from ball
    jump    Pad_Coll    ; Check for collision


Right_Side:
    load    v8 254      ; Set X direction to left
    load    v3 10       ; Set V3 to 10 in case right player misses ball
    move    v0 v7       ; Set V0 to V7 Y coord. of ball
    sub     v0 vd       ; Subtract position of paddle from ball


Pad_Coll:
    ske     vf 1        ; Skip next instruction if ball not above paddle
    jump    Ball_Lost

    load    v1 2        ; Set V1 to 02
    sub     v0 v1       ; Subtract V1 from V0
    ske     vf 1        ; Skip next instr. if ball not at top of paddle
    jump    Ball_Top    ; Ball at top of paddle

    sub     v0 v1       ; Subtract another 2 from V0
    ske     vf 1        ; Skip next instr. if ball not at middle of paddle
    jump    Pad_Hit     ; Ball in middle of paddle

    sub     v0 v1       ; Subtract another 2 from V0
    ske     vf 1        ; Skip next instr. if ball not at bottom of paddle
    jump    Ball_Bot    ; Ball at bottom of paddle


Ball_Lost:
    load    v0 32       ; Set lost ball beep delay
    loads   v0          ; Beep for lost ball

    call    Draw_Score  ; Erase previous score
    addr    ve v3       ; Add 1 or 10 to score depending on V3
    call    Draw_Score  ; Write new score

    load    v6 62       ; Set ball X coord. to right side
    ske     v3 1        ; Skip next instr. if right player got point
    load    v6 3        ; Set ball X coord. to left side
    load    v8 $fe      ; Set direction to left
    ske     v3 1        ; Skip next instr. if right player got point
    load    v8 2        ; Set direction to right
    jump    Big_Loop


Ball_Top:
    add     v9 $ff      ; Subtract 1 from V9, ball Y direction
    skne    v9 $fe      ; Skip next instr. if V9 != FE (-2)
    load    v9 $ff      ; Set V9=FF (-1)
    jump    Pad_Hit


Ball_Bot:
    add     v9 1        ; Add 1 to V9, ball Y direction
    skne    v9 2        ; Skip next instr. if V9 != 02
    load    v9 1        ; Set V9=01


Pad_Hit:
    load    v0 4        ; Set beep for paddle hit
    loads   v0          ; Sound beep

    add     v6 1
    skne    v6 64
    add     v6 254

    jump    Ball_Loop


Draw_Score:
    loadi   Score       ; Get address of Score
    bcd     ve          ; Stores in memory BCD representation of VE
    read    v2          ; Reads V0...V2 in memory, so the score
    ldspr   v1          ; I points to hex char in V1, so the 1st score char
    load    v4 $14      ; Set V4 to the X coord. to draw 1st score char
    load    v5 0        ; Set V5 to the Y coord. to draw 1st score char
    draw    v4 v5 5     ; Draw 8*5 sprite at (V4,V5) from M[I], so char V1
    add     v4 $15      ; Set X to the X coord. of 2nd score char
    ldspr   v2          ; I points to hex char in V2, so 2nd score char
    draw    v4 v5 5     ; Draw 8*5 sprite at (V4,V5) from M[I], so char V2
    rts


Paddle:
    .. $8080
    .. $8080
    .. $8080


Ball:
    .. $8000


Score:
    .. $0000
    .. $0000

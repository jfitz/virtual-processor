msg_z:	STRING	"Value is zero"
msg_nz:	STRING	"Value is not zero"

MAIN:	PUSH BYTE	0
	FLAGS BYTE
	POP BYTE
	ZERO JUMP	zero
	PUSH BYTE	msg_nz
	CALL	print_s
	CALL	print_nl
	JUMP	exit
zero:	PUSH BYTE	msg_z
	CALL	print_s
	CALL	print_nl
exit:	EXIT

address:	BYTE	0
print_s:	POP BYTE	@address
loop:	PUSH BYTE	@@address
	FLAGS BYTE
	ZERO RET
	OUT
	INC BYTE	@address
	JUMP	loop

newline:	BYTE	10
print_nl:	PUSH BYTE	@newline
	OUT
	RET

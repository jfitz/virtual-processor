message:	STRING	"Hello, world!"

MAIN:	PUSH BYTE	message
	CALL	print_s
	CALL	print_nl
	EXIT

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

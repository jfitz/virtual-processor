message	STRING	"Hello, world!"

MAIN	PUSH.B	message
	CALL	print_s
	CALL	print_nl
	EXIT

address	BYTE	0
print_s	POP.B	@address
loop	PUSH.B	@@address
	FLAGS.B
	Z:RET
	OUT
	INC.B	@address
	JUMP	loop

newline	BYTE	10
print_nl PUSH.B	@newline
	OUT
	RET

message	STRING	"Hello, world!"

MAIN	PUSH.B	message
	CALL.R	print_s
	CALL.R	print_nl
	EXIT

address	BYTE	0
print_s	POP.B	@address
loop	PUSH.B	@@address
	FLAGS.B
	RZ
	OUT
	INC.B	@address
	JUMP.R	loop

newline	BYTE	10
print_nl PUSH.B	@newline
	OUT
	RET

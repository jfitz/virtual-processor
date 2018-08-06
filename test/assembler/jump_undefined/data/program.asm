message	STRING	"Hello, world!"
address	BYTE	0
newline	BYTE	10

MAIN	PUSH.B	message
	POP.B	@address

loop	PUSH.B	@@address
	FLAGS.B
	Z:JUMP	printnl
	OUT
	INC.B	@address
	JUMP.A	loop_wrong

printnl	PUSH.B	@newline
	OUT
	EXIT

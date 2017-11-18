message	STRING	"Hello, world!"
address	BYTE	0
newline	BYTE	10

MAIN	PUSH.B	message
	POP.B	@address

loop	PUSH.B	@@address
	FLAGS.B
	JZ	printnl
	OUT.B
	INC.B	@address
	JUMP	loop

printnl	PUSH.B	@newline
	OUT.B
	EXIT

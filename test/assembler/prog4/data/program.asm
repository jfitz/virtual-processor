message	STRING	"Hello, world!"
address	BYTE	0
newline	BYTE	10

MAIN	PUSH.B	message
	POP.B	@address

loop	PUSH.B	@@address
	FLAGS.B
	JRZ	printnl
	OUT
	INC.B	@address
	JREL	loop

printnl	PUSH.B	@newline
	OUT
	EXIT

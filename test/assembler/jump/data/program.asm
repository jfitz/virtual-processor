message:	STRING	"Hello, world!"
address:	BYTE	0
newline:	BYTE	10

MAIN:	PUSH BYTE	message
	POP BYTE	@address

loop:	PUSH BYTE	@@address
	FLAGS BYTE
	ZERO JUMP	printnl
	OUT
	INC BYTE	@address
	JUMP	loop

printnl:	PUSH BYTE	@newline
	OUT
	EXIT

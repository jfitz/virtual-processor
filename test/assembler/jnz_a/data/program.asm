message	STRING	"Hello, world!"
address	BYTE	0
count	BYTE	0

MAIN	PUSH.B	message
	POP.B	@address
	PUSH.B	14
	POP.B	@count

loop	DEC.B	@count
	FLAGS.B	@count
	JNZ.A	print
	JUMP.A	printnl
print	PUSH.B	@@address
	OUT
	INC.B	@address
	JUMP.A	loop

newline	BYTE	10
printnl	PUSH.B	@newline
	OUT
	EXIT

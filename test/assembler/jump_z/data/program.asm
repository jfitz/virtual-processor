msg_z	STRING	"Value is zero"
msg_nz	STRING	"Value is not zero"

MAIN	PUSH.B	0
	FLAGS.B
	POP.B	0
	Z:JUMP	zero
	PUSH.B	msg_nz
	CALL	print_s
	CALL	print_nl
	JUMP	exit
zero	PUSH.B	msg_z
	CALL	print_s
	CALL	print_nl
exit	EXIT

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

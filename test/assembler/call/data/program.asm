message:	STRING	"Hello, world!"

# main program
MAIN:	PUSH BYTE	message
	CALL	print_s
	CALL	print_nl
	EXIT

# print a string
address:	BYTE	0
print_s:	POP BYTE	@address
loop:	PUSH BYTE	@@address
	FLAGS BYTE
	ZERO RET
	OUT
	INC BYTE	@address
	JUMP	loop

# print a newline
newline:	BYTE	10
print_nl:	PUSH BYTE	@newline
	OUT
	RET

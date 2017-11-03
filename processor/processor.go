/*
package main of vcpu
*/
package main

import (
	"fmt"
	"github.com/jfitz/virtual-processor/vputils"
	"os"
	"strings"
)

func check(e error) {
	if e != nil {
		panic(e)
	}
}

func showErrorAndStop(message string) {
	if message != "" {
		fmt.Println(message)
		os.Exit(1)
	}
}

func readHeader(f *os.File) string {
	bytes := []byte{}
	one_byte := make([]byte, 1)
	one_byte[0] = 1
	for one_byte[0] != 0 {
		_, err := f.Read(one_byte)
		check(err)
		if one_byte[0] != 0 {
			bytes = append(bytes, one_byte...)
		}
	}

	name := string(bytes)

	return name
}

func readTextTable(f *os.File) []vputils.NameValue {
	stx_byte := []byte{0x02}
	etx_byte := []byte{0x03}
	fs_byte := []byte{0x1c}
	rs_byte := []byte{0x1e}

	one_byte := make([]byte, 1)

	// read STX
	_, err := f.Read(one_byte)
	check(err)

	if one_byte[0] != stx_byte[0] {
		showErrorAndStop("Did not find STX")
	}

	// read until ETX
	bytes := []byte{}
	one_byte[0] = 0
	for one_byte[0] != etx_byte[0] {
		_, err := f.Read(one_byte)
		check(err)
		if one_byte[0] != etx_byte[0] {
			bytes = append(bytes, one_byte...)
		}
	}

	all_text := string(bytes)
	records := strings.Split(all_text, string(rs_byte))

	nameValues := []vputils.NameValue{}

	for _, record := range records {
		fields := strings.Split(record, string(fs_byte))
		if len(fields) == 2 {
			name := fields[0]
			value := fields[1]
			nameValue := vputils.NameValue{name, value}
			nameValues = append(nameValues, nameValue)
		}
	}

	return nameValues
}

func readBinaryBlock(f *os.File) []byte {
	count := make([]byte, 2)
	f.Read(count)
	countBytes := int(count[1])<<8 + int(count[0])

	code := make([]byte, countBytes)
	_, err := f.Read(code)
	check(err)

	checkCount := make([]byte, 2)
	f.Read(checkCount)
	checkCountBytes := int(checkCount[1])<<8 + int(checkCount[0])

	if checkCountBytes != countBytes {
		panic("Block count error")
	}

	return code
}

func executeCode(code []byte) {
	pc := 0
	stack := [1024]byte{}
	sp := 0

	fmt.Printf("Execution started at %04x\n", pc)
	halt := false
	for !halt {
		opcode := code[pc]

		switch opcode {
		case 0x00:
			// EXIT
			halt = true
			pc += 1
		case 0x40:
			// PUSH B V
			stack[sp] = code[pc+1]
			sp += 1
			if sp == len(stack) {
				fmt.Printf("Stack overflow at %04x\n", pc)
			}
			pc += 2
		case 0x51:
			// POP B A
		case 0x13:
			// OUT B S
			if sp == 0 {
				fmt.Printf("Stack underflow at %04x\n", pc)
			}
			sp -= 1
			c := stack[sp]
			fmt.Print(string(c))
			pc += 1
		default:
			// invalid opcode
			fmt.Printf("Invalid opcode %02x at %04x\n", opcode, pc)
			return
		}
	}

	fmt.Printf("Execution halted at %04x\n", pc)
}

func main() {
	args := os.Args[1:]

	if len(args) == 0 {
		fmt.Println("No module file specified")
		os.Exit(1)
	}

	moduleFile := args[0]
	fmt.Printf("Opening file '%s'\n", moduleFile)

	f, err := os.Open(moduleFile)
	check(err)

	defer f.Close()

	header := readHeader(f)
	if header != "module" {
		fmt.Println("Did not find module header")
		return
	}

	header = readHeader(f)
	if header != "properties" {
		fmt.Println("Did not find properties header")
		return
	}

	properties := readTextTable(f)

	for _, nameValue := range properties {
		fmt.Printf("%s: %s\n", nameValue.Name, nameValue.Value)
	}

	header = readHeader(f)
	if header != "code" {
		fmt.Println("Did not find code header")
		return
	}

	code := readBinaryBlock(f)

	fmt.Printf("Code length: %04x\n", len(code))

	header = readHeader(f)
	if header != "data" {
		fmt.Println("Did not find data header")
		return
	}

	data := readBinaryBlock(f)

	fmt.Printf("Data length: %04x\n", len(data))

	executeCode(code)
}

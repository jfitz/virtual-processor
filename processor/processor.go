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

func checkStack(sp int, stack []byte) {
	if sp == len(stack) {
		vputils.ShowErrorAndStop("Stack overflow")
	}
}

func executeCode(code []byte, data []byte) {
	pc := 0
	stack := make([]byte, 1024)
	sp := 0
	value := byte(0)
	codeAddress := byte(0)
	dataAddress := byte(0)

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
			// PUSH.B Value
			codeAddress = byte(pc + 1)
			value = code[codeAddress]
			stack[sp] = value
			sp += 1
			checkStack(sp, stack)
			pc += 2
		case 0x41:
			// PUSH.B Address
			codeAddress = byte(pc + 1)
			dataAddress = code[codeAddress]
			value = data[dataAddress]
			stack[sp] = value
			sp += 1
			checkStack(sp, stack)
			pc += 2
		case 0x51:
			// POP.B A
		case 0x13:
			// OUT.B
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
	vputils.Check(err)

	defer f.Close()

	header := vputils.ReadString(f)
	if header != "module" {
		fmt.Println("Did not find module header")
		return
	}

	header = vputils.ReadString(f)
	if header != "properties" {
		fmt.Println("Did not find properties header")
		return
	}

	properties := vputils.ReadTextTable(f)

	codeWidth := 0
	dataWidth := 0
	for _, nameValue := range properties {
		fmt.Printf("%s: %s\n", nameValue.Name, nameValue.Value)
		shortName := strings.Replace(nameValue.Name, " ", "", -1)
		if shortName == "CODEADDRESSWIDTH" {
			codeWidth = 1
		}
		if shortName == "DATAADDRESSWIDTH" {
			dataWidth = 1
		}
	}

	header = vputils.ReadString(f)
	if header != "code" {
		fmt.Println("Did not find code header")
		return
	}

	code := vputils.ReadBinaryBlock(f, codeWidth)

	fmt.Printf("Code length: %04x\n", len(code))

	header = vputils.ReadString(f)
	if header != "data" {
		fmt.Println("Did not find data header")
		return
	}

	data := vputils.ReadBinaryBlock(f, dataWidth)

	fmt.Printf("Data length: %04x\n", len(data))

	executeCode(code, data)
}

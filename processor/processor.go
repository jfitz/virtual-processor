/*
package main of vcpu
*/
package main

import (
	"errors"
	"flag"
	"fmt"
	"github.com/jfitz/virtual-processor/vputils"
	"os"
	"strconv"
	"strings"
)

type stack []byte

func (s stack) push(v byte) stack {
	return append(s, v)
}

func (s stack) top() (byte, error) {
	if len(s) == 0 {
		return 0, errors.New("Stack underflow")
	}

	last := len(s) - 1
	return s[last], nil
}

func (s stack) pop() (stack, error) {
	if len(s) == 0 {
		return s, errors.New("Stack underflow")
	}

	last := len(s) - 1
	return s[:last], nil
}

func (s stack) toppop() (byte, stack, error) {
	if len(s) == 0 {
		return 0, s, errors.New("Stack underflow")
	}

	last := len(s) - 1
	return s[last], s[:last], nil
}

type instructionDefinition struct {
	Name        string
	TargetSize  string
	AddressMode string
	JumpMode    string
}

func (def instructionDefinition) to_s() string {
	s := def.Name

	if def.TargetSize != "" {
		s += "."
		s += def.TargetSize
	}

	return s
}

type instructionTable map[byte]instructionDefinition

func defineInstructions() instructionTable {
	instructionDefinitions := make(instructionTable)
	instructionDefinitions[0x00] = instructionDefinition{"EXIT", "", "", ""}
	instructionDefinitions[0x40] = instructionDefinition{"PUSH", "B", "V", ""}
	instructionDefinitions[0x41] = instructionDefinition{"PUSH", "B", "D", ""}
	instructionDefinitions[0x42] = instructionDefinition{"PUSH", "B", "I", ""}
	instructionDefinitions[0x51] = instructionDefinition{"POP", "B", "D", ""}
	instructionDefinitions[0x08] = instructionDefinition{"OUT", "", "S", ""}
	instructionDefinitions[0x11] = instructionDefinition{"FLAGS", "B", "D", ""}
	instructionDefinitions[0x12] = instructionDefinition{"FLAGS", "B", "I", ""}
	instructionDefinitions[0x13] = instructionDefinition{"FLAGS", "B", "S", ""}
	instructionDefinitions[0x21] = instructionDefinition{"INC", "B", "D", ""}
	instructionDefinitions[0x22] = instructionDefinition{"INC", "B", "I", ""}
	instructionDefinitions[0x90] = instructionDefinition{"JUMP", "", "", "A"}
	instructionDefinitions[0x92] = instructionDefinition{"JZ", "", "", "A"}

	return instructionDefinitions
}

func executeCode(module vputils.Module, startAddress vputils.Address, trace bool, instructionDefinitions instructionTable) {
	bytesPerCodeAddress := 1
	bytesPerDataAddress := 1
	flags := [1]bool{false}
	pc := startAddress
	vStack := make(stack, 0)

	if trace {
		fmt.Printf("Execution started at %04x\n", pc.ByteValue())
	}

	code := module.Code
	data := module.Data
	halt := false
	for !halt {
		opcode, err := code.GetByte(pc)
		vputils.CheckPrintAndExit(err, "at PC "+pc.ToString())

		def := instructionDefinitions[opcode]
		value := byte(0)
		value_s := ""
		dataAddress := vputils.Address{[]byte{}}
		dataAddress1 := vputils.Address{[]byte{}}
		jumpAddress := vputils.Address{[]byte{}}

		instructionSize := 1
		targetSize := 0

		if def.TargetSize == "B" {
			targetSize = 1
		}
		if def.TargetSize == "W" {
			targetSize = 2
		}
		if def.TargetSize == "L" {
			targetSize = 4
		}
		if def.TargetSize == "F" {
			targetSize = 8
		}

		if def.AddressMode == "V" {
			instructionSize += targetSize

			value = module.GetImmediateByte(pc)
			value_s = fmt.Sprintf("%02X", value)
		}
		if def.AddressMode == "D" {
			instructionSize += bytesPerDataAddress

			dataAddress = module.GetDirectAddress(pc)

			value, _ = module.GetDirectByte(pc)
			value_s = fmt.Sprintf("%02X", value)
		}
		if def.AddressMode == "I" {
			instructionSize += bytesPerDataAddress

			dataAddress1 = module.GetDirectAddress(pc)
			dataAddress = module.GetIndirectAddress(pc)
			value, _ = module.GetIndirectByte(pc)
			value_s = fmt.Sprintf("%02X", value)
		}

		if def.JumpMode == "A" {
			instructionSize += bytesPerCodeAddress

			codeAddress := pc.AddByte(1)
			jumpAddr, _ := code.GetByte(codeAddress)
			jumpAddress = vputils.Address{[]byte{jumpAddr}}
		}
		if def.JumpMode == "R" {
			instructionSize += bytesPerCodeAddress

			codeAddress := pc.AddByte(1)
			jumpAddr, _ := code.GetByte(codeAddress)
			jumpAddress = vputils.Address{[]byte{jumpAddr}}
		}

		if trace {
			text := def.to_s()
			line := fmt.Sprintf("%s: %02X %s", pc.ToString(), opcode, text)
			if !dataAddress1.Empty() {
				line += " @@" + dataAddress1.ToString()
			}
			if !dataAddress.Empty() {
				line += " @" + dataAddress.ToString()
			}
			if len(value_s) > 0 {
				line += " =" + value_s
			}
			if !jumpAddress.Empty() {
				line += " >" + jumpAddress.ToString()
			}
			fmt.Println(line)
		}

		switch opcode {
		case 0x00:
			// EXIT
			halt = true

			pc = pc.AddByte(instructionSize)

		case 0x40:
			// PUSH.B immediate value
			vStack = vStack.push(value)

			pc = pc.AddByte(instructionSize)

		case 0x41:
			// PUSH.B direct address
			vStack = vStack.push(value)

			pc = pc.AddByte(instructionSize)

		case 0x42:
			// PUSH.B indirect address
			vStack = vStack.push(value)

			pc = pc.AddByte(instructionSize)

		case 0x51:
			// POP.B direct address
			value, vStack, err = vStack.toppop()
			vputils.CheckAndPanic(err)

			err = data.PutByte(dataAddress, value)
			vputils.CheckAndPanic(err)

			pc = pc.AddByte(instructionSize)

		case 0x08:
			// OUT (implied stack)
			value, vStack, err = vStack.toppop()
			vputils.CheckAndPanic(err)

			fmt.Print(string(value))

			if trace {
				fmt.Println()
			}

			pc = pc.AddByte(instructionSize)

		case 0x11:
			// FLAGS.B direct address
			flags[0] = value == 0

			pc = pc.AddByte(instructionSize)

		case 0x12:
			// FLAGS.B indirect address
			flags[0] = value == 0

			pc = pc.AddByte(instructionSize)

		case 0x13:
			// FLAGS.B (implied stack)
			value, err = vStack.top()
			vputils.CheckAndPanic(err)

			flags[0] = value == 0

			pc = pc.AddByte(instructionSize)

		case 0x21:
			// INC.B direct address
			value += 1

			err = data.PutByte(dataAddress, value)
			vputils.CheckAndPanic(err)

			pc = pc.AddByte(instructionSize)

		case 0x22:
			// INC.B indirect address
			value += 1

			err = data.PutByte(dataAddress, value)
			vputils.CheckAndPanic(err)

			pc = pc.AddByte(instructionSize)

		case 0x90:
			// JUMP
			pc = jumpAddress

		case 0x92:
			// JZ
			if flags[0] {
				pc = jumpAddress
			} else {
				pc = pc.AddByte(instructionSize)
			}

		default:
			// invalid opcode
			fmt.Printf("Invalid opcode %02x at %s\n", opcode, pc.ToString())
			return
		}
	}

	if trace {
		fmt.Println("Execution halted at " + pc.ToString())
	}
}

func read(moduleFile string) (vputils.Module, error) {
	f, err := os.Open(moduleFile)
	vputils.CheckAndExit(err)

	defer f.Close()

	header := vputils.ReadString(f)
	if header != "module" {
		return vputils.Module{}, errors.New("Did not find module header")
	}

	header = vputils.ReadString(f)
	if header != "properties" {
		return vputils.Module{}, errors.New("Did not find properties header")
	}

	properties := vputils.ReadTextTable(f)

	codeAddressWidth := 0
	dataAddressWidth := 0
	for _, nameValue := range properties {
		shortName := strings.Replace(nameValue.Name, " ", "", -1)
		if shortName == "CODEADDRESSWIDTH" {
			codeAddressWidth = 1
		}
		if shortName == "DATAADDRESSWIDTH" {
			dataAddressWidth = 1
		}
	}

	header = vputils.ReadString(f)
	if header != "exports" {
		return vputils.Module{}, errors.New("Did not find exports header")
	}

	exports := vputils.ReadTextTable(f)

	header = vputils.ReadString(f)
	if header != "code" {
		return vputils.Module{}, errors.New("Did not find code header")
	}

	code := vputils.ReadBinaryBlock(f, codeAddressWidth)

	header = vputils.ReadString(f)
	if header != "data" {
		return vputils.Module{}, errors.New("Did not find data header")
	}

	data := vputils.ReadBinaryBlock(f, dataAddressWidth)

	return vputils.Module{properties, code, exports, data, codeAddressWidth, dataAddressWidth}, nil
}

func main() {
	startSymbolPtr := flag.String("start", "MAIN", "Start execution at symbol.")
	tracePtr := flag.Bool("trace", false, "Display trace during execution.")

	flag.Parse()

	startSymbol := *startSymbolPtr
	trace := *tracePtr

	args := flag.Args()

	if len(args) == 0 {
		fmt.Println("No module file specified")
		os.Exit(1)
	}

	moduleFile := args[0]

	module, err := read(moduleFile)
	vputils.CheckAndExit(err)

	code := module.Code
	exports := module.Exports
	codeAddressWidth := module.CodeAddressWidth

	startAddressFound := false
	startAddressInt := 0
	for _, nameValue := range exports {
		if nameValue.Name == startSymbol {
			startAddressFound = true
			startAddressInt, err = strconv.Atoi(nameValue.Value)
			vputils.CheckPrintAndExit(err, "Invalid start address")
		}
	}

	if !startAddressFound {
		fmt.Println("Starting symbol " + startSymbol + " not found")
		os.Exit(2)
	}

	startAddress := vputils.MakeAddress(startAddressInt, codeAddressWidth)

	if int(startAddress.ByteValue()) >= len(code) {
		fmt.Println("Starting address " + startAddress.ToString() + " is not valid")
		os.Exit(2)
	}

	instructionDefinitions := defineInstructions()

	executeCode(module, startAddress, trace, instructionDefinitions)
}

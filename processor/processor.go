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

func (def instructionDefinition) toString() string {
	s := def.Name

	if len(def.TargetSize) > 0 {
		s += "."
		s += def.TargetSize
	}

	if len(def.JumpMode) > 0 {
		s += "."
		s += def.JumpMode
	}

	return s
}

type instructionTable map[byte]instructionDefinition

func defineInstructions() instructionTable {
	instructionDefinitions := make(instructionTable)
	instructionDefinitions[0x00] = instructionDefinition{"EXIT", "", "", ""}
	instructionDefinitions[0x60] = instructionDefinition{"PUSH", "B", "V", ""}
	instructionDefinitions[0x61] = instructionDefinition{"PUSH", "B", "D", ""}
	instructionDefinitions[0x62] = instructionDefinition{"PUSH", "B", "I", ""}
	instructionDefinitions[0x81] = instructionDefinition{"POP", "B", "D", ""}
	instructionDefinitions[0x08] = instructionDefinition{"OUT", "", "S", ""}
	instructionDefinitions[0x11] = instructionDefinition{"FLAGS", "B", "D", ""}
	instructionDefinitions[0x12] = instructionDefinition{"FLAGS", "B", "I", ""}
	instructionDefinitions[0x13] = instructionDefinition{"FLAGS", "B", "S", ""}
	instructionDefinitions[0x21] = instructionDefinition{"INC", "B", "D", ""}
	instructionDefinitions[0x22] = instructionDefinition{"INC", "B", "I", ""}
	instructionDefinitions[0xD0] = instructionDefinition{"JUMP", "", "", "A"}
	instructionDefinitions[0xD2] = instructionDefinition{"JZ", "", "", "A"}
	instructionDefinitions[0xE0] = instructionDefinition{"JUMP", "", "", "R"}
	instructionDefinitions[0xE2] = instructionDefinition{"JZ", "", "", "R"}

	return instructionDefinitions
}

func (def instructionDefinition) calcInstructionSize() int {
	return 1
}

func (def instructionDefinition) calcTargetSize() int {
	targetSize := 0

	if def.TargetSize == "B" {
		targetSize = 1
	}
	if def.TargetSize == "I16" {
		targetSize = 2
	}
	if def.TargetSize == "I32" {
		targetSize = 4
	}
	if def.TargetSize == "I64" {
		targetSize = 8
	}
	if def.TargetSize == "F32" {
		targetSize = 4
	}
	if def.TargetSize == "F64" {
		targetSize = 8
	}

	return targetSize
}

func executeCode(module vputils.Module, startAddress vputils.Address, trace bool, instructionDefinitions instructionTable) {
	flags := [1]bool{false}
	module.SetPC(startAddress)
	vStack := make(stack, 0)

	if trace {
		fmt.Printf("Execution started at %04x\n", module.PCByteValue())
	}

	code := module.Code
	data := module.Data
	halt := false
	for !halt {
		pc := module.PC()
		opcode, err := code.GetByte(pc)
		vputils.CheckPrintAndExit(err, "at PC "+pc.ToString())

		def := instructionDefinitions[opcode]
		value := byte(0)
		value_s := ""
		dataAddress := vputils.Address{[]byte{}}
		dataAddress1 := vputils.Address{[]byte{}}
		jumpAddress := vputils.Address{[]byte{}}
		offset := byte(0)
		offset_s := ""

		instructionSize := def.calcInstructionSize()
		targetSize := def.calcTargetSize()

		if def.AddressMode == "V" {
			value = module.ImmediateByte()
			value_s = fmt.Sprintf("%02X", value)

			instructionSize += targetSize
		}
		if def.AddressMode == "D" {
			dataAddress = module.DirectAddress()

			value, _ = module.DirectByte()
			value_s = fmt.Sprintf("%02X", value)

			instructionSize += dataAddress.Size()
		}
		if def.AddressMode == "I" {
			dataAddress1 = module.DirectAddress()
			dataAddress = module.IndirectAddress()
			value, _ = module.IndirectByte()
			value_s = fmt.Sprintf("%02X", value)

			instructionSize += dataAddress1.Size()
		}

		if def.JumpMode == "A" {
			jumpAddress = module.DirectAddress()

			instructionSize += jumpAddress.Size()
		}
		if def.JumpMode == "R" {
			offset = module.ImmediateByte()
			offset_i := int(offset)
			if offset_i > 127 {
				offset_i = offset_i - 256
			}
			offset_s = strconv.Itoa(offset_i)
			jumpAddress = pc.AddByte(offset_i)

			instructionSize += 1
		}

		if trace {
			text := def.toString()
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
			if len(offset_s) > 0 {
				line += " " + offset_s
			}
			if !jumpAddress.Empty() {
				line += " >" + jumpAddress.ToString()
			}
			fmt.Println(line)
		}

		newpc := pc
		switch opcode {
		case 0x00:
			// EXIT
			halt = true

			newpc = pc.AddByte(instructionSize)

		case 0x60:
			// PUSH.B immediate value
			vStack = vStack.push(value)

			newpc = pc.AddByte(instructionSize)

		case 0x61:
			// PUSH.B direct address
			vStack = vStack.push(value)

			newpc = pc.AddByte(instructionSize)

		case 0x62:
			// PUSH.B indirect address
			vStack = vStack.push(value)

			newpc = pc.AddByte(instructionSize)

		case 0x81:
			// POP.B direct address
			value, vStack, err = vStack.toppop()
			vputils.CheckAndPanic(err)

			err = data.PutByte(dataAddress, value)
			vputils.CheckAndPanic(err)

			newpc = pc.AddByte(instructionSize)

		case 0x08:
			// OUT (implied stack)
			value, vStack, err = vStack.toppop()
			vputils.CheckAndPanic(err)

			fmt.Print(string(value))

			if trace {
				fmt.Println()
			}

			newpc = pc.AddByte(instructionSize)

		case 0x11:
			// FLAGS.B direct address
			flags[0] = value == 0

			newpc = pc.AddByte(instructionSize)

		case 0x12:
			// FLAGS.B indirect address
			flags[0] = value == 0

			newpc = pc.AddByte(instructionSize)

		case 0x13:
			// FLAGS.B (implied stack)
			value, err = vStack.top()
			vputils.CheckAndPanic(err)

			flags[0] = value == 0

			newpc = pc.AddByte(instructionSize)

		case 0x21:
			// INC.B direct address
			value += 1

			err = data.PutByte(dataAddress, value)
			vputils.CheckAndPanic(err)

			newpc = pc.AddByte(instructionSize)

		case 0x22:
			// INC.B indirect address
			value += 1

			err = data.PutByte(dataAddress, value)
			vputils.CheckAndPanic(err)

			newpc = pc.AddByte(instructionSize)

		case 0xD0:
			// JUMP.A
			newpc = jumpAddress

		case 0xD2:
			// JZ.A
			if flags[0] {
				newpc = jumpAddress
			} else {
				newpc = pc.AddByte(instructionSize)
			}

		case 0xE0:
			// JUMP.R
			newpc = jumpAddress

		case 0xE2:
			// JZ.R
			if flags[0] {
				newpc = jumpAddress
			} else {
				newpc = pc.AddByte(instructionSize)
			}

		default:
			// invalid opcode
			fmt.Printf("Invalid opcode %02x at %s\n", opcode, pc.ToString())
			return
		}

		module.SetPC(newpc)
	}

	if trace {
		pc := module.PC()
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

	return vputils.Module{
		Properties:       properties,
		Code:             code,
		Exports:          exports,
		Data:             data,
		CodeAddressWidth: codeAddressWidth,
		DataAddressWidth: dataAddressWidth,
	}, nil
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

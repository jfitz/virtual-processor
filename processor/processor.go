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

type byteStack []byte

func (s byteStack) push(v byte) byteStack {
	return append(s, v)
}

func (s byteStack) top() (byte, error) {
	count := 1
	if len(s) < count {
		return 0, errors.New("Stack underflow")
	}

	last := len(s) - count
	return s[last], nil
}

func (s byteStack) pop() (byteStack, error) {
	count := 1
	if len(s) < count {
		return s, errors.New("Stack underflow")
	}

	last := len(s) - count
	return s[:last], nil
}

func (s byteStack) toppop(count int) ([]byte, byteStack, error) {
	if len(s) < count {
		return []byte{}, s, errors.New("Stack underflow")
	}

	last := len(s) - count
	return s[last:], s[:last], nil
}

type instructionDefinition struct {
	Name        string
	TargetType  string
	AddressMode string
	JumpMode    string
}

func (def instructionDefinition) toString() string {
	s := def.Name

	if len(def.TargetType) > 0 {
		s += "."
		s += def.TargetType
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
	instructionDefinitions[0x01] = instructionDefinition{"KCALL", "", "", ""}

	instructionDefinitions[0x60] = instructionDefinition{"PUSH", "B", "V", ""}
	instructionDefinitions[0x61] = instructionDefinition{"PUSH", "B", "D", ""}
	instructionDefinitions[0x62] = instructionDefinition{"PUSH", "B", "I", ""}

	instructionDefinitions[0x64] = instructionDefinition{"PUSH", "I16", "V", ""}
	instructionDefinitions[0x65] = instructionDefinition{"PUSH", "I16", "D", ""}
	instructionDefinitions[0x66] = instructionDefinition{"PUSH", "I16", "I", ""}

	instructionDefinitions[0x79] = instructionDefinition{"PUSH", "STR", "D", ""}

	instructionDefinitions[0x81] = instructionDefinition{"POP", "B", "D", ""}
	instructionDefinitions[0x08] = instructionDefinition{"OUT", "", "S", ""}

	instructionDefinitions[0x11] = instructionDefinition{"FLAGS", "B", "D", ""}
	instructionDefinitions[0x12] = instructionDefinition{"FLAGS", "B", "I", ""}
	instructionDefinitions[0x13] = instructionDefinition{"FLAGS", "B", "S", ""}

	instructionDefinitions[0x21] = instructionDefinition{"INC", "B", "D", ""}
	instructionDefinitions[0x22] = instructionDefinition{"INC", "B", "I", ""}
	instructionDefinitions[0x31] = instructionDefinition{"DEC", "B", "D", ""}
	instructionDefinitions[0x32] = instructionDefinition{"DEC", "B", "I", ""}

	instructionDefinitions[0xD0] = instructionDefinition{"JUMP", "", "", "A"}
	instructionDefinitions[0xD1] = instructionDefinition{"JNZ", "", "", "A"}
	instructionDefinitions[0xD2] = instructionDefinition{"JZ", "", "", "A"}

	instructionDefinitions[0xE0] = instructionDefinition{"JUMP", "", "", "R"}
	instructionDefinitions[0xE1] = instructionDefinition{"JNZ", "", "", "R"}
	instructionDefinitions[0xE2] = instructionDefinition{"JZ", "", "", "R"}

	instructionDefinitions[0xD4] = instructionDefinition{"CALL", "", "", "A"}
	instructionDefinitions[0xD5] = instructionDefinition{"CNZ", "", "", "A"}
	instructionDefinitions[0xD6] = instructionDefinition{"CZ", "", "", "A"}

	instructionDefinitions[0xE4] = instructionDefinition{"CALL", "", "", "R"}
	instructionDefinitions[0xE5] = instructionDefinition{"CNZ", "", "", "R"}
	instructionDefinitions[0xE6] = instructionDefinition{"CZ", "", "", "R"}

	instructionDefinitions[0xD8] = instructionDefinition{"RET", "", "", ""}
	instructionDefinitions[0xD9] = instructionDefinition{"RNZ", "", "", ""}
	instructionDefinitions[0xDA] = instructionDefinition{"RZ", "", "", ""}

	instructionDefinitions[0xA0] = instructionDefinition{"ADD", "B", "", ""}
	instructionDefinitions[0xA1] = instructionDefinition{"SUB", "B", "", ""}
	instructionDefinitions[0xA2] = instructionDefinition{"MUL", "B", "", ""}
	instructionDefinitions[0xA3] = instructionDefinition{"DIV", "B", "", ""}

	instructionDefinitions[0xC0] = instructionDefinition{"AND", "B", "", ""}
	instructionDefinitions[0xC1] = instructionDefinition{"OR", "B", "", ""}
	instructionDefinitions[0xC3] = instructionDefinition{"CMP", "B", "", ""}

	return instructionDefinitions
}

func (def instructionDefinition) calcInstructionSize() int {
	return 1
}

func (def instructionDefinition) calcTargetSize() int {
	targetSize := 0

	if def.TargetType == "B" {
		targetSize = 1
	}
	if def.TargetType == "I16" {
		targetSize = 2
	}
	if def.TargetType == "I32" {
		targetSize = 4
	}
	if def.TargetType == "I64" {
		targetSize = 8
	}
	if def.TargetType == "F32" {
		targetSize = 4
	}
	if def.TargetType == "F64" {
		targetSize = 8
	}

	return targetSize
}

func pop_string(vStack byteStack) (string, byteStack) {
	// pop size of name
	counts, vStack, err := vStack.toppop(1)
	vputils.CheckAndExit(err)
	count := int(counts[0])

	// pop name of function
	bytes := []byte{}
	s := ""
	for i := 0; i < count; i++ {
		bytes, vStack, err = vStack.toppop(1)
		vputils.CheckAndExit(err)
		if bytes[0] != 0 {
			s += string(bytes[0])
		}
	}

	return s, vStack
}

func kernelCall(vStack byteStack) byteStack {
	fname, vStack := pop_string(vStack)

	// dispatch to function
	bytes := []byte{}
	s := ""
	err := errors.New("")
	switch fname {
	case "out_b":
		bytes, vStack, err = vStack.toppop(1)
		vputils.CheckAndPanic(err)

		fmt.Print(string(bytes[0]))

	case "out_s":
		s, vStack = pop_string(vStack)

		fmt.Print(s)

	default:
		err = errors.New("Unknown kernel call to function '" + fname + "'")
		vputils.CheckAndExit(err)
	}

	// return to module
	return vStack
}

func executeCode(module vputils.Module, startAddress vputils.Address, trace bool, instructionDefinitions instructionTable) {
	flags := [1]bool{false}
	module.SetPC(startAddress)
	vStack := make(byteStack, 0) // value stack

	if trace {
		fmt.Println("Execution started at ", startAddress.ToString())
	}

	code := module.Code
	data := module.Data
	halt := false
	for !halt {
		pc := module.PC()
		opcode, err := code.GetByte(pc)
		vputils.CheckPrintAndExit(err, "at PC "+pc.ToString())

		def := instructionDefinitions[opcode]
		bytes := []byte{0}
		bytes1 := []byte{}
		bytes2 := []byte{}
		value_s := ""
		dataAddress := vputils.Address{[]byte{}}
		dataAddress1 := vputils.Address{[]byte{}}
		jumpAddress := vputils.Address{[]byte{}}
		offset_s := ""

		instructionSize := def.calcInstructionSize()
		targetSize := def.calcTargetSize()

		if def.AddressMode == "V" {
			switch def.TargetType {
			case "B":
				bytes = module.ImmediateByte()
				value_s = fmt.Sprintf("%02X", bytes[0])
			case "I16":
				bytes = module.ImmediateInt()
				value_s = fmt.Sprintf("%02X%02X", bytes[1], bytes[0])
			}
			instructionSize += targetSize
		}

		if def.AddressMode == "D" {
			dataAddress = module.DirectAddress()
			bytes[0], _ = module.DirectByte()
			value_s = fmt.Sprintf("%02X", bytes[0])

			instructionSize += dataAddress.Size()
		}

		if def.AddressMode == "I" {
			dataAddress1 = module.DirectAddress()
			dataAddress = module.IndirectAddress()
			bytes[0], _ = module.IndirectByte()
			value_s = fmt.Sprintf("%02X", bytes)

			instructionSize += dataAddress1.Size()
		}

		if def.JumpMode == "A" {
			jumpAddress = module.DirectAddress()

			instructionSize += jumpAddress.Size()
		}

		if def.JumpMode == "R" {
			bytes = module.ImmediateByte()
			offset_i := int(bytes[0])
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

		case 0x01:
			// KCALL - kernel call
			newpc = pc.AddByte(instructionSize)

			vStack = kernelCall(vStack)

		case 0x08:
			// OUT (implied stack)
			bytes, vStack, err = vStack.toppop(1)
			vputils.CheckAndPanic(err)

			fmt.Print(string(bytes[0]))

			if trace {
				fmt.Println()
			}

			newpc = pc.AddByte(instructionSize)

		case 0x11:
			// FLAGS.B direct address
			flags[0] = bytes[0] == 0

			newpc = pc.AddByte(instructionSize)

		case 0x12:
			// FLAGS.B indirect address
			flags[0] = bytes[0] == 0

			newpc = pc.AddByte(instructionSize)

		case 0x13:
			// FLAGS.B (implied stack)
			bytes[0], err = vStack.top()
			vputils.CheckAndPanic(err)

			flags[0] = bytes[0] == 0

			newpc = pc.AddByte(instructionSize)

		case 0x21:
			// INC.B direct address
			bytes[0] += 1

			err = data.PutByte(dataAddress, bytes[0])
			vputils.CheckAndPanic(err)

			newpc = pc.AddByte(instructionSize)

		case 0x22:
			// INC.B indirect address
			bytes[0] += 1

			err = data.PutByte(dataAddress, bytes[0])
			vputils.CheckAndPanic(err)

			newpc = pc.AddByte(instructionSize)

		case 0x31:
			// DEC.B direct address
			bytes[0] -= 1

			err = data.PutByte(dataAddress, bytes[0])
			vputils.CheckAndPanic(err)

			newpc = pc.AddByte(instructionSize)

		case 0x32:
			// DEC.B indirect address
			bytes[0] -= 1

			err = data.PutByte(dataAddress, bytes[0])
			vputils.CheckAndPanic(err)

			newpc = pc.AddByte(instructionSize)

		case 0x60:
			// PUSH.B immediate value
			vStack = vStack.push(bytes[0])

			newpc = pc.AddByte(instructionSize)

		case 0x61:
			// PUSH.B direct address
			vStack = vStack.push(bytes[0])

			newpc = pc.AddByte(instructionSize)

		case 0x62:
			// PUSH.B indirect address
			vStack = vStack.push(bytes[0])

			newpc = pc.AddByte(instructionSize)

		case 0x64:
			// PUSH.I16 immediate value
			vStack = vStack.push(bytes[1])
			vStack = vStack.push(bytes[0])

			newpc = pc.AddByte(instructionSize)

		case 0x65:
			// PUSH.I16 direct address
			vStack = vStack.push(bytes[1])
			vStack = vStack.push(bytes[0])

			newpc = pc.AddByte(instructionSize)

		case 0x66:
			// PUSH.I16 indirect address
			vStack = vStack.push(bytes[1])
			vStack = vStack.push(bytes[0])

			newpc = pc.AddByte(instructionSize)

		case 0x79:
			// PUSH.STR direct address
			s := ""
			address := dataAddress
			b := byte(1)

			for b != 0 {
				b, err = data.GetByte(address)
				vputils.CheckAndExit(err)
				c := string(b)
				s += c
				address = address.AddByte(1)
			}

			count := len(s)
			// push bytes to stack in reverse order
			for i := range s {
				c := s[count-i-1]
				b = byte(c)
				vStack = vStack.push(b)
			}
			b = byte(count)
			vStack = vStack.push(b)

			newpc = pc.AddByte(instructionSize)

		case 0x81:
			// POP.B direct address
			bytes, vStack, err = vStack.toppop(1)
			vputils.CheckAndExit(err)

			err = data.PutByte(dataAddress, bytes[0])
			vputils.CheckAndPanic(err)

			newpc = pc.AddByte(instructionSize)

		case 0xA0:
			// ADD.B
			bytes1, vStack, err = vStack.toppop(1)
			vputils.CheckAndExit(err)

			bytes2, vStack, err = vStack.toppop(1)
			vputils.CheckAndExit(err)

			value := bytes1[0] + bytes2[0]
			vStack = vStack.push(value)

			newpc = pc.AddByte(instructionSize)

		case 0xA1:
			// SUB.B
			bytes1, vStack, err = vStack.toppop(1)
			vputils.CheckAndExit(err)

			bytes2, vStack, err = vStack.toppop(1)
			vputils.CheckAndExit(err)

			value := bytes1[0] - bytes2[0]
			vStack = vStack.push(value)

			newpc = pc.AddByte(instructionSize)

		case 0xA2:
			// MUL.B
			bytes1, vStack, err = vStack.toppop(1)
			vputils.CheckAndExit(err)

			bytes2, vStack, err = vStack.toppop(1)
			vputils.CheckAndExit(err)

			value := bytes1[0] * bytes2[0]
			vStack = vStack.push(value)

			newpc = pc.AddByte(instructionSize)

		case 0xA3:
			// DIV.B
			bytes1, vStack, err = vStack.toppop(1)
			vputils.CheckAndExit(err)

			bytes2, vStack, err = vStack.toppop(1)
			vputils.CheckAndExit(err)

			value := bytes1[0] / bytes2[0]
			vStack = vStack.push(value)

			newpc = pc.AddByte(instructionSize)

		case 0xC0:
			// AND.B
			bytes1, vStack, err = vStack.toppop(1)
			vputils.CheckAndExit(err)

			bytes2, vStack, err = vStack.toppop(1)
			vputils.CheckAndExit(err)

			value := bytes1[0] & bytes2[0]
			vStack = vStack.push(value)

			newpc = pc.AddByte(instructionSize)

		case 0xC1:
			// OR.B
			bytes1, vStack, err = vStack.toppop(1)
			vputils.CheckAndExit(err)

			bytes2, vStack, err = vStack.toppop(1)
			vputils.CheckAndExit(err)

			value := bytes1[0] | bytes2[0]
			vStack = vStack.push(value)

			newpc = pc.AddByte(instructionSize)

		case 0xC3:
			// CMP.B
			bytes1, vStack, err = vStack.toppop(1)
			vputils.CheckAndExit(err)

			bytes2, vStack, err = vStack.toppop(1)
			vputils.CheckAndExit(err)

			value := bytes1[0] - bytes2[0]

			flags[0] = value == 0

			newpc = pc.AddByte(instructionSize)

		case 0xD0:
			// JUMP.A
			newpc = jumpAddress

		case 0xD1:
			// JNZ.A
			if !flags[0] {
				newpc = jumpAddress
			} else {
				newpc = pc.AddByte(instructionSize)
			}

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

		case 0xE1:
			// JNZ.R
			if !flags[0] {
				newpc = jumpAddress
			} else {
				newpc = pc.AddByte(instructionSize)
			}

		case 0xE2:
			// JZ.R
			if flags[0] {
				newpc = jumpAddress
			} else {
				newpc = pc.AddByte(instructionSize)
			}

		case 0xD4:
			// CALL.A
			newpc = jumpAddress
			retpc := pc.AddByte(instructionSize)
			module.Push(retpc)

		case 0xD5:
			// CNZ.A
			if !flags[0] {
				newpc = jumpAddress
				retpc := pc.AddByte(instructionSize)
				module.Push(retpc)
			} else {
				newpc = pc.AddByte(instructionSize)
			}

		case 0xD6:
			// CZ.A
			if flags[0] {
				newpc = jumpAddress
				retpc := pc.AddByte(instructionSize)
				module.Push(retpc)
			} else {
				newpc = pc.AddByte(instructionSize)
			}

		case 0xE4:
			// CALL.R
			newpc = jumpAddress
			retpc := pc.AddByte(instructionSize)
			module.Push(retpc)

		case 0xE5:
			// CNZ.R
			if !flags[0] {
				newpc = jumpAddress
				retpc := pc.AddByte(instructionSize)
				module.Push(retpc)
			} else {
				newpc = pc.AddByte(instructionSize)
			}

		case 0xE6:
			// CZ.R
			if flags[0] {
				newpc = jumpAddress
				retpc := pc.AddByte(instructionSize)
				module.Push(retpc)
			} else {
				newpc = pc.AddByte(instructionSize)
			}

		case 0xD8:
			// RET
			newpc, err = module.TopPop()
			vputils.CheckAndExit(err)

		case 0xD9:
			// RNZ
			if !flags[0] {
				newpc, err = module.TopPop()
				vputils.CheckAndExit(err)
			} else {
				newpc = pc.AddByte(instructionSize)
			}

		case 0xDA:
			// RZ
			if flags[0] {
				newpc, err = module.TopPop()
				vputils.CheckAndExit(err)
			} else {
				newpc = pc.AddByte(instructionSize)
			}

		default:
			// invalid opcode
			fmt.Printf("Invalid opcode %02x at %s\n", opcode, pc.ToString())
			return
		}

		if trace {
			stack := ""
			for _, v := range vStack {
				stack += fmt.Sprintf(" %02X", v)
			}
			fmt.Println("Value stack:" + stack)
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

	module := vputils.Module{
		Properties:       properties,
		Code:             code,
		Exports:          exports,
		Data:             data,
		CodeAddressWidth: codeAddressWidth,
		DataAddressWidth: dataAddressWidth,
	}

	module.Init()

	return module, nil
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

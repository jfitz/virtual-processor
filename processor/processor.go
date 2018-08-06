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

func (stack byteStack) pushByte(v byte) byteStack {
	return append(stack, v)
}

func reverseBytes(bs []byte) []byte {
	last := len(bs) - 1

	for i := 0; i < len(bs)/2; i++ {
		bs[i], bs[last-i] = bs[last-i], bs[i]
	}

	return bs
}

func (stack byteStack) pushBytes(vs []byte) byteStack {
	bs := reverseBytes(vs)
	return append(stack, bs...)
}

func (stack byteStack) topByte() (byte, error) {
	count := 1
	if len(stack) < count {
		return 0, errors.New("Stack underflow")
	}

	last := len(stack) - count
	return stack[last], nil
}

func (stack byteStack) popByte(count int) ([]byte, byteStack, error) {
	if len(stack) < count {
		return []byte{}, stack, errors.New("Stack underflow")
	}

	last := len(stack) - count
	return stack[last:], stack[:last], nil
}

func (stack byteStack) pushString(s string) byteStack {
	bs := []byte(s)
	stack = stack.pushBytes(bs)
	b := byte(len(s))
	stack = stack.pushByte(b)

	return stack
}

func (stack byteStack) popString() (string, byteStack) {
	// pop size of name
	counts, stack, err := stack.popByte(1)
	vputils.CheckAndExit(err)
	count := int(counts[0])

	// pop bytes that make the string
	bytes := []byte{}
	s := ""
	for i := 0; i < count; i++ {
		bytes, stack, err = stack.popByte(1)
		vputils.CheckAndExit(err)
		if bytes[0] != 0 {
			s += string(bytes[0])
		}
	}

	return s, stack
}

type instructionDefinition struct {
	Name        string
	TargetType  string
	AddressMode string
}

func (def instructionDefinition) toString() string {
	s := def.Name

	if len(def.TargetType) > 0 {
		s += "."
		s += def.TargetType
	}

	return s
}

type instructionTable map[byte]instructionDefinition

func defineInstructions() instructionTable {
	instructionDefinitions := make(instructionTable)

	instructionDefinitions[0x00] = instructionDefinition{"EXIT", "", ""}
	instructionDefinitions[0x01] = instructionDefinition{"KCALL", "", ""}

	instructionDefinitions[0x60] = instructionDefinition{"PUSH", "B", "V"}
	instructionDefinitions[0x61] = instructionDefinition{"PUSH", "B", "D"}
	instructionDefinitions[0x62] = instructionDefinition{"PUSH", "B", "I"}

	instructionDefinitions[0x64] = instructionDefinition{"PUSH", "I16", "V"}
	instructionDefinitions[0x65] = instructionDefinition{"PUSH", "I16", "D"}
	instructionDefinitions[0x66] = instructionDefinition{"PUSH", "I16", "I"}

	instructionDefinitions[0x79] = instructionDefinition{"PUSH", "STR", "D"}

	instructionDefinitions[0x81] = instructionDefinition{"POP", "B", "D"}
	instructionDefinitions[0x08] = instructionDefinition{"OUT", "", "S"}

	instructionDefinitions[0x11] = instructionDefinition{"FLAGS", "B", "D"}
	instructionDefinitions[0x12] = instructionDefinition{"FLAGS", "B", "I"}
	instructionDefinitions[0x13] = instructionDefinition{"FLAGS", "B", "S"}

	instructionDefinitions[0x21] = instructionDefinition{"INC", "B", "D"}
	instructionDefinitions[0x22] = instructionDefinition{"INC", "B", "I"}
	instructionDefinitions[0x31] = instructionDefinition{"DEC", "B", "D"}
	instructionDefinitions[0x32] = instructionDefinition{"DEC", "B", "I"}

	instructionDefinitions[0xD0] = instructionDefinition{"JUMP", "", ""}
	instructionDefinitions[0xD1] = instructionDefinition{"CALL", "", ""}
	instructionDefinitions[0xD2] = instructionDefinition{"RET", "", ""}

	instructionDefinitions[0xA0] = instructionDefinition{"ADD", "B", ""}
	instructionDefinitions[0xA1] = instructionDefinition{"SUB", "B", ""}
	instructionDefinitions[0xA2] = instructionDefinition{"MUL", "B", ""}
	instructionDefinitions[0xA3] = instructionDefinition{"DIV", "B", ""}

	instructionDefinitions[0xC0] = instructionDefinition{"AND", "B", ""}
	instructionDefinitions[0xC1] = instructionDefinition{"OR", "B", ""}
	instructionDefinitions[0xC3] = instructionDefinition{"CMP", "B", ""}

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

func kernelCall(vStack byteStack) byteStack {
	fname, vStack := vStack.popString()

	// dispatch to function
	bytes := []byte{}
	s := ""
	err := errors.New("")
	switch fname {
	case "out_b":
		bytes, vStack, err = vStack.popByte(1)
		vputils.CheckAndPanic(err)

		fmt.Print(string(bytes[0]))

	case "out_s":
		s, vStack = vStack.popString()

		fmt.Print(s)

	default:
		err = errors.New("Unknown kernel call to function '" + fname + "'")
		vputils.CheckAndExit(err)
	}

	// return to module
	return vStack
}

func getConditionAndOpcode(code vputils.Vector, pc vputils.Address) ([]byte, byte, error) {
	conditionals := []byte{}
	opcode := byte(0)
	err := errors.New("")

	newpc := pc
	my_byte, err := code.GetByte(newpc)

	has_conditional := true

	for has_conditional {
		if my_byte >= 0xE0 && my_byte <= 0xEF {
			conditionals = append(conditionals, my_byte)
			newpc := newpc.AddByte(1)
			my_byte, err = code.GetByte(newpc)
		} else {
			opcode = my_byte
			has_conditional = false
		}
	}

	return conditionals, opcode, err
}

func evaluateConditionals(conditionals []byte, flags []bool) bool {
	execute := true
	stack := []bool{}

	for _, conditional := range conditionals {
		switch conditional {
		case 0xE0:
			stack = append(stack, flags[0])
		}
	}

	if len(stack) == 1 {
		execute = stack[0]
	}

	return execute
}

func conditionalsToString(conditionals []byte) string {
	result := ""

	for _, conditional := range conditionals {
		switch conditional {
		case 0xE0:
			result += "Z:"
		}
	}

	return result
}

func executeCode(module vputils.Module, startAddress vputils.Address, trace bool, instructionDefinitions instructionTable) error {
	// initialize virtual processor
	flags := []bool{false}
	vStack := make(byteStack, 0) // value stack

	// initialize module
	err := module.SetPC(startAddress)
	if err != nil {
		s := fmt.Sprintf("Invalid start address %s for main: %s", startAddress.ToString(), err.Error())
		return errors.New(s)
	}

	// trace
	if trace {
		fmt.Println("Execution started at ", startAddress.ToString())
	}

	code := module.Code
	data := module.Data
	halt := false
	for !halt {
		pc := module.PC()
		conditionals, opcode, err := getConditionAndOpcode(code, pc)
		vputils.CheckPrintAndExit(err, "at PC "+pc.ToString())

		newpc := pc.AddByte(len(conditionals))

		execute := true

		if len(conditionals) > 0 {
			err = module.SetPC(newpc)

			execute = evaluateConditionals(conditionals, flags)
		}

		instructionSize := len(conditionals)

		// get opcode definition
		def := instructionDefinitions[opcode]

		// bytes for opcode
		bytes := []byte{0}
		bytes1 := []byte{}
		bytes2 := []byte{}
		value_s := ""

		// addresses for opcode
		dataAddress := vputils.Address{[]byte{}}
		dataAddress1 := vputils.Address{[]byte{}}
		jumpAddress := vputils.Address{[]byte{}}
		offset_s := ""

		instructionSize += def.calcInstructionSize()
		targetSize := def.calcTargetSize()

		// decode immediate value
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

		// decode memory target
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

		// decode jump target
		if opcode == 0xD0 || opcode == 0xD1 {
			jumpAddress = module.DirectAddress()

			instructionSize += jumpAddress.Size()
		}

		// trace opcode and arguments
		if trace {
			text := def.toString()
			condi_s := conditionalsToString(conditionals)
			line := fmt.Sprintf("%s: %02X %s%s", pc.ToString(), opcode, condi_s, text)
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
			if flags[0] {
				line += " Z"
			} else {
				line += " z"
			}
			fmt.Println(line)
		}

		// execute opcode
		switch opcode {
		case 0x00:
			// EXIT
			halt = true

			// newpc = pc.AddByte(instructionSize)

		case 0x01:
			// KCALL - kernel call
			newpc = pc.AddByte(instructionSize)

			vStack = kernelCall(vStack)

		case 0x08:
			// OUT (implied stack)
			bytes, vStack, err = vStack.popByte(1)
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
			bytes[0], err = vStack.topByte()
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
			vStack = vStack.pushBytes(bytes)

			newpc = pc.AddByte(instructionSize)

		case 0x61:
			// PUSH.B direct address
			vStack = vStack.pushBytes(bytes)

			newpc = pc.AddByte(instructionSize)

		case 0x62:
			// PUSH.B indirect address
			vStack = vStack.pushBytes(bytes)

			newpc = pc.AddByte(instructionSize)

		case 0x64:
			// PUSH.I16 immediate value
			vStack = vStack.pushBytes(bytes)

			newpc = pc.AddByte(instructionSize)

		case 0x65:
			// PUSH.I16 direct address
			vStack = vStack.pushBytes(bytes)

			newpc = pc.AddByte(instructionSize)

		case 0x66:
			// PUSH.I16 indirect address
			vStack = vStack.pushBytes(bytes)

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

			vStack = vStack.pushString(s)

			newpc = pc.AddByte(instructionSize)

		case 0x81:
			// POP.B direct address
			bytes, vStack, err = vStack.popByte(1)
			vputils.CheckAndExit(err)

			err = data.PutByte(dataAddress, bytes[0])
			vputils.CheckAndPanic(err)

			newpc = pc.AddByte(instructionSize)

		case 0xA0:
			// ADD.B
			bytes1, vStack, err = vStack.popByte(1)
			vputils.CheckAndExit(err)

			bytes2, vStack, err = vStack.popByte(1)
			vputils.CheckAndExit(err)

			value := bytes1[0] + bytes2[0]
			vStack = vStack.pushByte(value)

			newpc = pc.AddByte(instructionSize)

		case 0xA1:
			// SUB.B
			bytes1, vStack, err = vStack.popByte(1)
			vputils.CheckAndExit(err)

			bytes2, vStack, err = vStack.popByte(1)
			vputils.CheckAndExit(err)

			value := bytes1[0] - bytes2[0]
			vStack = vStack.pushByte(value)

			newpc = pc.AddByte(instructionSize)

		case 0xA2:
			// MUL.B
			bytes1, vStack, err = vStack.popByte(1)
			vputils.CheckAndExit(err)

			bytes2, vStack, err = vStack.popByte(1)
			vputils.CheckAndExit(err)

			value := bytes1[0] * bytes2[0]
			vStack = vStack.pushByte(value)

			newpc = pc.AddByte(instructionSize)

		case 0xA3:
			// DIV.B
			bytes1, vStack, err = vStack.popByte(1)
			vputils.CheckAndExit(err)

			bytes2, vStack, err = vStack.popByte(1)
			vputils.CheckAndExit(err)

			value := bytes1[0] / bytes2[0]
			vStack = vStack.pushByte(value)

			newpc = pc.AddByte(instructionSize)

		case 0xC0:
			// AND.B
			bytes1, vStack, err = vStack.popByte(1)
			vputils.CheckAndExit(err)

			bytes2, vStack, err = vStack.popByte(1)
			vputils.CheckAndExit(err)

			value := bytes1[0] & bytes2[0]
			vStack = vStack.pushByte(value)

			newpc = pc.AddByte(instructionSize)

		case 0xC1:
			// OR.B
			bytes1, vStack, err = vStack.popByte(1)
			vputils.CheckAndExit(err)

			bytes2, vStack, err = vStack.popByte(1)
			vputils.CheckAndExit(err)

			value := bytes1[0] | bytes2[0]
			vStack = vStack.pushByte(value)

			newpc = pc.AddByte(instructionSize)

		case 0xC3:
			// CMP.B
			bytes1, vStack, err = vStack.popByte(1)
			vputils.CheckAndExit(err)

			bytes2, vStack, err = vStack.popByte(1)
			vputils.CheckAndExit(err)

			value := bytes1[0] - bytes2[0]

			flags[0] = value == 0

			newpc = pc.AddByte(instructionSize)

		case 0xD0:
			// JUMP
			newpc = jumpAddress

		case 0xD1:
			// CALL
			newpc = jumpAddress
			retpc := pc.AddByte(instructionSize)
			module.Push(retpc)

		case 0xD2:
			// RET
			if execute {
				newpc, err = module.TopPop()
				vputils.CheckAndExit(err)
			} else {
				newpc = pc.AddByte(instructionSize)
			}

		default:
			// invalid opcode
			s := fmt.Sprintf("Invalid opcode %02x at %s\n", opcode, pc.ToString())
			return errors.New(s)
		}

		// trace stack
		if trace {
			stack := ""
			for _, v := range vStack {
				stack += fmt.Sprintf(" %02X", v)
			}
			fmt.Println("Value stack:" + stack)
		}

		// advance to next instruction
		err = module.SetPC(newpc)
		if err != nil {
			s := fmt.Sprintf("Invalid address %s for PC in main: %s", newpc.ToString(), err.Error())
			return errors.New(s)
		}
	}

	// trace
	if trace {
		pc := module.PC()
		fmt.Println("Execution halted at " + pc.ToString())
	}

	return nil
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

	err = executeCode(module, startAddress, trace, instructionDefinitions)
	vputils.CheckAndExit(err)
}

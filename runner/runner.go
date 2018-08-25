/*
Package main of virtual CPU runner
*/
package main

import (
	"errors"
	"flag"
	"fmt"
	"github.com/jfitz/virtual-processor/module"
	"github.com/jfitz/virtual-processor/vputils"
	"os"
	"strconv"
)

// --------------------
// opcode definition
// --------------------
type opcodeDefinition struct {
	Name        string
	TargetType  string
	AddressMode string
}

// --------------------
func (def opcodeDefinition) toString() string {
	s := def.Name

	if len(def.TargetType) > 0 {
		s += "."
		s += def.TargetType
	}

	return s
}

// --------------------
func (def opcodeDefinition) opcodeSize() int {
	return 1
}

// --------------------
func (def opcodeDefinition) targetSize() int {
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

// --------------------
// --------------------

// --------------------
// opcodeTable
// --------------------
type opcodeTable map[byte]opcodeDefinition

// --------------------
func defineOpcodes() opcodeTable {
	opcodeDefinitions := make(opcodeTable)

	opcodeDefinitions[0x00] = opcodeDefinition{"NOP", "", ""}
	opcodeDefinitions[0x04] = opcodeDefinition{"EXIT", "", ""}
	opcodeDefinitions[0x05] = opcodeDefinition{"KCALL", "", ""}

	opcodeDefinitions[0x60] = opcodeDefinition{"PUSH", "B", "V"}
	opcodeDefinitions[0x61] = opcodeDefinition{"PUSH", "B", "D"}
	opcodeDefinitions[0x62] = opcodeDefinition{"PUSH", "B", "I"}

	opcodeDefinitions[0x64] = opcodeDefinition{"PUSH", "I16", "V"}
	opcodeDefinitions[0x65] = opcodeDefinition{"PUSH", "I16", "D"}
	opcodeDefinitions[0x66] = opcodeDefinition{"PUSH", "I16", "I"}

	opcodeDefinitions[0x79] = opcodeDefinition{"PUSH", "STR", "D"}

	opcodeDefinitions[0x80] = opcodeDefinition{"POP", "B", "V"}
	opcodeDefinitions[0x81] = opcodeDefinition{"POP", "B", "D"}
	opcodeDefinitions[0x08] = opcodeDefinition{"OUT", "", "S"}

	opcodeDefinitions[0x11] = opcodeDefinition{"FLAGS", "B", "D"}
	opcodeDefinitions[0x12] = opcodeDefinition{"FLAGS", "B", "I"}
	opcodeDefinitions[0x13] = opcodeDefinition{"FLAGS", "B", "S"}

	opcodeDefinitions[0x21] = opcodeDefinition{"INC", "B", "D"}
	opcodeDefinitions[0x22] = opcodeDefinition{"INC", "B", "I"}
	opcodeDefinitions[0x31] = opcodeDefinition{"DEC", "B", "D"}
	opcodeDefinitions[0x32] = opcodeDefinition{"DEC", "B", "I"}

	opcodeDefinitions[0xD0] = opcodeDefinition{"JUMP", "", ""}
	opcodeDefinitions[0xD1] = opcodeDefinition{"CALL", "", ""}
	opcodeDefinitions[0xD2] = opcodeDefinition{"RET", "", ""}

	opcodeDefinitions[0xA0] = opcodeDefinition{"ADD", "B", ""}
	opcodeDefinitions[0xA1] = opcodeDefinition{"SUB", "B", ""}
	opcodeDefinitions[0xA2] = opcodeDefinition{"MUL", "B", ""}
	opcodeDefinitions[0xA3] = opcodeDefinition{"DIV", "B", ""}

	opcodeDefinitions[0xC0] = opcodeDefinition{"AND", "B", ""}
	opcodeDefinitions[0xC1] = opcodeDefinition{"OR", "B", ""}
	opcodeDefinitions[0xC3] = opcodeDefinition{"CMP", "B", ""}

	return opcodeDefinitions
}

// --------------------
// --------------------

func getConditionals(code vputils.Vector, pc vputils.Address) (module.Conditionals, error) {
	conditionals := module.Conditionals{}
	err := errors.New("")

	newpc := pc
	myByte, err := code.GetByte(newpc)

	hasConditional := true

	for hasConditional {
		if myByte >= 0xE0 && myByte <= 0xEF {
			conditionals = append(conditionals, myByte)
			newpc = newpc.AddByte(1)
			myByte, err = code.GetByte(newpc)
		} else {
			hasConditional = false
		}
	}

	return conditionals, err
}

func kernelCall(vStack vputils.ByteStack) vputils.ByteStack {
	fname, vStack := vStack.PopString()

	// dispatch to function
	bytes := []byte{}
	s := ""
	err := errors.New("")

	switch fname {

	case "out_b":
		bytes, vStack, err = vStack.PopByte(1)
		vputils.CheckAndPanic(err)

		fmt.Print(string(bytes[0]))

	case "out_s":
		s, vStack = vStack.PopString()

		fmt.Print(s)

	default:
		err = errors.New("Unknown kernel call to function '" + fname + "'")
		vputils.CheckAndExit(err)

	}

	// return to module
	return vStack
}

func decodeInstruction(opcode byte, def opcodeDefinition, mod module.Module) module.InstructionDefinition {
	// bytes for opcode
	bytes := []byte{0}

	// addresses for opcode
	dataAddress := vputils.Address{[]byte{}, 0}
	dataAddress1 := vputils.Address{[]byte{}, 0}
	jumpAddress := vputils.Address{[]byte{}, 0}
	valueStr := ""

	instructionSize := def.opcodeSize()
	targetSize := def.targetSize()

	// decode immediate value
	if def.AddressMode == "V" {

		switch def.TargetType {

		case "B":
			bytes = mod.ImmediateByte()
			valueStr = fmt.Sprintf("%02X", bytes[0])

		case "I16":
			bytes = mod.ImmediateInt()
			valueStr = fmt.Sprintf("%02X%02X", bytes[1], bytes[0])

		}

		instructionSize += targetSize
	}

	// decode memory target
	if def.AddressMode == "D" {
		dataAddress = mod.DirectAddress()
		bytes[0], _ = mod.DirectByte()
		valueStr = fmt.Sprintf("%02X", bytes[0])

		instructionSize += dataAddress.NumBytes()
	}

	if def.AddressMode == "I" {
		dataAddress1 = mod.DirectAddress()
		dataAddress = mod.IndirectAddress()
		bytes[0], _ = mod.IndirectByte()
		valueStr = fmt.Sprintf("%02X", bytes[0])

		instructionSize += dataAddress1.NumBytes()
	}

	// decode jump/call target
	if opcode == 0xD0 || opcode == 0xD1 {
		jumpAddress = mod.DirectAddress()

		instructionSize += jumpAddress.NumBytes()
	}

	instruction := module.InstructionDefinition{dataAddress1, dataAddress, instructionSize, jumpAddress, bytes, valueStr}

	return instruction
}

func traceOpcode(pc vputils.Address, opcode byte, def opcodeDefinition, flags module.FlagsGroup, conditionals module.Conditionals, instruction module.InstructionDefinition) string {
	dataAddress1 := instruction.Address1
	dataAddress := instruction.Address
	jumpAddress := instruction.JumpAddress
	valueStr := instruction.ValueStr

	line := fmt.Sprintf("%s: ", pc.ToString())

	text := def.toString()
	if len(conditionals) > 0 {
		condiStr := conditionals.ToString()
		condiByteStr := conditionals.ToByteString()
		line += fmt.Sprintf("%s %02X %s:%s", condiByteStr, opcode, condiStr, text)
	} else {
		line += fmt.Sprintf("%02X %s", opcode, text)
	}

	if !dataAddress1.Empty() {
		line += " @@" + dataAddress1.ToString()
	}
	if !dataAddress.Empty() {
		line += " @" + dataAddress.ToString()
	}

	if len(valueStr) > 0 {
		line += " =" + valueStr
	}

	if !jumpAddress.Empty() {
		line += " >" + jumpAddress.ToString()
	}

	line += flags.ToString()

	return line
}

func traceValueStack(stack vputils.ByteStack) string {
	line := "Value stack:"

	s := stack.ToByteString()

	if len(s) > 0 {
		line += " " + s
	}

	return line
}

func traceHalt(pc vputils.Address) string {
	line := "Execution halted at " + pc.ToString()

	return line
}

func executeCode(mod module.Module, startAddress vputils.Address, trace bool, opcodeDefinitions opcodeTable) error {
	// initialize virtual processor
	flags := module.FlagsGroup{false, false, false}
	vStack := make(vputils.ByteStack, 0) // value stack

	// initialize module
	err := mod.SetPC(startAddress)
	if err != nil {
		s := fmt.Sprintf("Invalid start address %s for main: %s", startAddress.ToString(), err.Error())
		return errors.New(s)
	}

	// trace
	if trace {
		fmt.Println("Execution started at ", startAddress.ToString())
	}

	halt := false

	for !halt {
		pc := mod.PC()
		conditionals, err := getConditionals(mod.Code, pc)
		vputils.CheckPrintAndExit(err, "at PC "+pc.ToString())

		opcodePC := pc.AddByte(len(conditionals))
		execute := true

		if len(conditionals) > 0 {
			err = mod.SetPC(opcodePC)
			if err != nil {
				return err
			}

			execute, err = conditionals.Evaluate(flags)
			if err != nil {
				return err
			}
		}

		opcode, err := mod.Code.GetByte(opcodePC)
		vputils.CheckPrintAndExit(err, "at PC "+pc.ToString())

		// get opcode definition
		def := opcodeDefinitions[opcode]

		instruction := decodeInstruction(opcode, def, mod)

		// trace opcode and arguments
		if trace {
			line := traceOpcode(pc, opcode, def, flags, conditionals, instruction)
			fmt.Println(line)
		}

		syscall := byte(0)
		vStack, flags, syscall, err = mod.ExecuteOpcode(opcode, vStack, instruction, execute, flags, trace)

		switch syscall {

		case 0x04:
			halt = true

		case 0x05:
			vStack = kernelCall(vStack)

		}

		// trace value stack
		if trace {
			line := traceValueStack(vStack)
			fmt.Println(line)
		}
	}

	// trace
	if trace {
		line := traceHalt(mod.PC())
		fmt.Println(line)
	}

	return nil
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

	mod, err := module.Read(moduleFile)
	vputils.CheckAndExit(err)

	exports := mod.Exports
	codeAddressWidth := mod.CodeAddressWidth

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

	startAddress, err := vputils.MakeAddress(startAddressInt, codeAddressWidth, len(mod.Code))
	vputils.CheckAndExit(err)

	opcodeDefinitions := defineOpcodes()

	err = executeCode(mod, startAddress, trace, opcodeDefinitions)
	vputils.CheckAndExit(err)
}

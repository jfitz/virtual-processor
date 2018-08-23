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
	"strings"
)

// --------------------
// instruction definition
// --------------------
type instructionDefinition struct {
	Name        string
	TargetType  string
	AddressMode string
}

// --------------------
func (def instructionDefinition) toString() string {
	s := def.Name

	if len(def.TargetType) > 0 {
		s += "."
		s += def.TargetType
	}

	return s
}

// --------------------
func (def instructionDefinition) calcInstructionSize() int {
	return 1
}

// --------------------
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

// --------------------
// --------------------

// --------------------
// instructionTable
// --------------------
type instructionTable map[byte]instructionDefinition

// --------------------
func defineInstructions() instructionTable {
	instructionDefinitions := make(instructionTable)

	instructionDefinitions[0x00] = instructionDefinition{"NOP", "", ""}
	instructionDefinitions[0x04] = instructionDefinition{"EXIT", "", ""}
	instructionDefinitions[0x05] = instructionDefinition{"KCALL", "", ""}

	instructionDefinitions[0x60] = instructionDefinition{"PUSH", "B", "V"}
	instructionDefinitions[0x61] = instructionDefinition{"PUSH", "B", "D"}
	instructionDefinitions[0x62] = instructionDefinition{"PUSH", "B", "I"}

	instructionDefinitions[0x64] = instructionDefinition{"PUSH", "I16", "V"}
	instructionDefinitions[0x65] = instructionDefinition{"PUSH", "I16", "D"}
	instructionDefinitions[0x66] = instructionDefinition{"PUSH", "I16", "I"}

	instructionDefinitions[0x79] = instructionDefinition{"PUSH", "STR", "D"}

	instructionDefinitions[0x80] = instructionDefinition{"POP", "B", "V"}
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

// --------------------
// --------------------

func getConditionAndOpcode(code vputils.Vector, pc vputils.Address) ([]byte, byte, error) {
	condiBytes := []byte{}
	opcode := byte(0)
	err := errors.New("")

	newpc := pc
	myByte, err := code.GetByte(newpc)

	hasConditional := true

	for hasConditional {
		if myByte >= 0xE0 && myByte <= 0xEF {
			condiBytes = append(condiBytes, myByte)
			newpc = newpc.AddByte(1)
			myByte, err = code.GetByte(newpc)
		} else {
			opcode = myByte
			hasConditional = false
		}
	}

	return condiBytes, opcode, err
}

func evaluateConditionals(condiBytes []byte, flags module.FlagsGroup) (bool, error) {
	execute := true
	stack := make(vputils.BoolStack, 0)

	for _, condiByte := range condiBytes {
		switch condiByte {
		case 0xE0:
			stack = stack.Push(flags.Zero)
		case 0xE8:
			top, stack, err := stack.Pop()
			if err != nil {
				return false, err
			}
			stack = stack.Push(!top)
		default:
			return false, errors.New("Invalid conditional")
		}
	}

	if len(stack) > 1 {
		return false, errors.New("Invalid conditionals")
	}

	if len(stack) == 1 {
		exe, err := stack.Top()
		if err != nil {
			return false, err
		}
		execute = exe
	}

	return execute, nil
}

func encodeConditional(condiByte byte) (string, error) {
	condiString := ""

	switch condiByte {
	case 0xE0:
		condiString = "Z"
	case 0xE8:
		condiString = "NOT"
	default:
		return "", errors.New("Invalid conditional code")
	}

	return condiString, nil
}

func encodeConditionals(condiBytes []byte) ([]string, error) {
	condiStrings := []string{}

	for _, condiByte := range condiBytes {
		condiString, err := encodeConditional(condiByte)
		if err != nil {
			return condiStrings, err
		}
		condiStrings = append(condiStrings, condiString)
	}

	return condiStrings, nil
}

func conditionalsToString(condiBytes []byte) (string, error) {
	condiStrings, err := encodeConditionals(condiBytes)
	if err != nil {
		return "", err
	}

	result := strings.Join(condiStrings, ".")

	return result, nil
}

func executeCode(mod module.Module, startAddress vputils.Address, trace bool, instructionDefinitions instructionTable) error {
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
		condiBytes, opcode, err := getConditionAndOpcode(mod.Code, pc)
		vputils.CheckPrintAndExit(err, "at PC "+pc.ToString())

		execute := true

		if len(condiBytes) > 0 {
			opcodePC := pc.AddByte(len(condiBytes))
			err = mod.SetPC(opcodePC)
			if err != nil {
				return err
			}

			execute, err = evaluateConditionals(condiBytes, flags)
			if err != nil {
				return err
			}
		}

		// get opcode definition
		def := instructionDefinitions[opcode]

		// bytes for opcode
		bytes := []byte{0}

		// addresses for opcode
		dataAddress := vputils.Address{[]byte{}, 0}
		dataAddress1 := vputils.Address{[]byte{}, 0}
		jumpAddress := vputils.Address{[]byte{}, 0}
		valueStr := ""

		instructionSize := def.calcInstructionSize()
		targetSize := def.calcTargetSize()

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

		// trace opcode and arguments
		if trace {
			text := def.toString()
			condiStr, err := conditionalsToString(condiBytes)

			if err != nil {
				fmt.Println(err.Error())
			}

			line := ""

			if len(condiStr) > 0 {
				line = fmt.Sprintf("%s: % 02X %02X %s:%s", pc.ToString(), condiBytes, opcode, condiStr, text)
			} else {
				line = fmt.Sprintf("%s: %02X %s", pc.ToString(), opcode, text)
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

			if flags.Zero {
				line += " Z"
			} else {
				line += " z"
			}

			fmt.Println(line)
		}

		vStack, flags, halt, err = mod.ExecuteOpcode(opcode, vStack, dataAddress, instructionSize, jumpAddress, bytes, execute, flags, trace)

		// trace stack
		if trace {
			stack := ""
			for _, v := range vStack {
				stack += fmt.Sprintf(" %02X", v)
			}
			fmt.Println("Value stack:" + stack)
		}
	}

	// trace
	if trace {
		pc := mod.PC()
		fmt.Println("Execution halted at " + pc.ToString())
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

	instructionDefinitions := defineInstructions()

	err = executeCode(mod, startAddress, trace, instructionDefinitions)
	vputils.CheckAndExit(err)
}

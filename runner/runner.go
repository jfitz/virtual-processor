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

func outCall(vStack vputils.ByteStack, trace bool) vputils.ByteStack {
	bytes, vStack, err := vStack.PopByte(1)
	vputils.CheckAndPanic(err)

	fmt.Print(string(bytes[0]))

	if trace {
		fmt.Println()
	}

	return vStack
}

func decodeInstruction(opcode byte, def module.OpcodeDefinition, mod module.Module) module.InstructionDefinition {
	fullOpcode := []byte{opcode}

	// working bytes for opcode
	workBytes := []byte{}

	// addresses for opcode
	dataAddress := vputils.Address{[]byte{}, 0}
	dataAddress1 := vputils.Address{[]byte{}, 0}
	jumpAddress := vputils.Address{[]byte{}, 0}
	valueStr := ""

	instructionSize := def.OpcodeSize()
	targetSize := def.TargetSize()

	err := errors.New("")

	// decode immediate value
	if def.AddressMode == "V" {

		switch def.Width {

		case "BYTE":
			workBytes, err = mod.ImmediateByte()
			vputils.CheckAndExit(err)

			valueStr = fmt.Sprintf("%02X", workBytes[0])

		case "I16":
			workBytes, err = mod.ImmediateInt()
			vputils.CheckAndExit(err)

			valueStr = fmt.Sprintf("%02X%02X", workBytes[1], workBytes[0])

		}

		fullOpcode = append(fullOpcode, workBytes...)
		instructionSize += targetSize
	}

	// decode memory target
	if def.AddressMode == "D" {
		dataAddress, err = mod.DirectAddress()
		vputils.CheckAndExit(err)

		fullOpcode = append(fullOpcode, dataAddress.Bytes...)

		buffer, err := mod.DirectByte()
		vputils.CheckAndExit(err)

		workBytes = append(workBytes, buffer)
		valueStr = fmt.Sprintf("%02X", buffer)

		instructionSize += dataAddress.NumBytes()
	}

	if def.AddressMode == "I" {
		dataAddress1, err = mod.DirectAddress()
		vputils.CheckAndExit(err)

		fullOpcode = append(fullOpcode, dataAddress1.Bytes...)
		workBytes = append(workBytes, dataAddress.Bytes...)

		dataAddress, err = mod.IndirectAddress()
		vputils.CheckAndExit(err)

		buffer, err := mod.IndirectByte()
		vputils.CheckAndExit(err)

		workBytes = append(workBytes, buffer)
		valueStr = fmt.Sprintf("%02X", buffer)

		instructionSize += dataAddress1.NumBytes()
	}

	// decode jump/call target
	if opcode == 0xD0 || opcode == 0xD1 {
		jumpAddress, err = mod.DirectAddress()
		vputils.CheckAndExit(err)

		fullOpcode = append(fullOpcode, jumpAddress.Bytes...)
		instructionSize += jumpAddress.NumBytes()
	}

	instruction := module.InstructionDefinition{fullOpcode, dataAddress1, dataAddress, instructionSize, jumpAddress, workBytes, valueStr}

	return instruction
}

func traceOpcode(pc vputils.Address, opcode byte, opcodeDef module.OpcodeDefinition, flags module.FlagsGroup, conditionals module.Conditionals, instruction module.InstructionDefinition) string {
	dataAddress1 := instruction.Address1
	dataAddress := instruction.Address
	jumpAddress := instruction.JumpAddress
	valueStr := instruction.ValueStr

	line := fmt.Sprintf("%s: ", pc.ToString())

	opcodeStr := instruction.ToByteString()

	text := opcodeDef.ToString()
	if len(conditionals) > 0 {
		condiStr := conditionals.ToString()
		condiByteStr := conditionals.ToByteString()
		line += fmt.Sprintf("%s%s%s %s", condiByteStr, opcodeStr, condiStr, text)
	} else {
		line += fmt.Sprintf("%s%s", opcodeStr, text)
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

func executeCode(mod module.Module, startAddress vputils.Address, trace bool, opcodeDefinitions module.OpcodeTable) error {
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
		pc1 := mod.PC()

		// get conditionals (if any)
		conditionals, err := mod.GetConditionals()
		vputils.CheckPrintAndExit(err, "at PC "+pc1.ToString())

		// evaluate conditionals
		execute, err := conditionals.Evaluate(flags)
		if err != nil {
			return err
		}

		// get the opcode
		pc2 := mod.PC()
		opcode, err := mod.GetOpcode()
		vputils.CheckPrintAndExit(err, "at PC "+pc2.ToString())

		// get opcode definition
		def := opcodeDefinitions[opcode]

		// get instruction definition (opcode and arguments)
		instruction := decodeInstruction(opcode, def, mod)

		// display instruction
		if trace {
			line := traceOpcode(pc1, opcode, def, flags, conditionals, instruction)
			fmt.Println(line)
		}

		// execute instruction
		syscall := byte(0)
		vStack, flags, syscall, err = mod.ExecuteOpcode(opcode, vStack, instruction, execute, flags)

		// process the requested runner call
		// these are handled here, not in the opcode processor
		switch syscall {

		case 0x04:
			halt = true

		case 0x05:
			vStack = kernelCall(vStack)

		case 0x08:
			vStack = outCall(vStack, trace)

		}

		// display value stack
		if trace {
			line := traceValueStack(vStack)
			fmt.Println(line)
		}
	}

	// display halt information
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

	startAddress, err := vputils.MakeAddress(startAddressInt, codeAddressWidth, len(mod.CodePage.Contents))
	vputils.CheckAndExit(err)

	opcodeDefinitions := module.DefineOpcodes()

	err = executeCode(mod, startAddress, trace, opcodeDefinitions)
	vputils.CheckAndExit(err)
}

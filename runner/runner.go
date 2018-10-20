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

func traceOpcode(pc vputils.Address, opcode byte, opcodeDef module.MnemonicTargetWidthAddressMode, flags module.FlagsGroup, conditionals module.Conditionals, instruction module.InstructionDefinition) string {
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

func executeCode(mod module.Module, proc module.Processor, startAddress vputils.Address, trace bool, opcodeDefinitions module.ByteToMnemonic) error {
	// initialize virtual processor
	flags := module.FlagsGroup{false, false, false}
	vStack := make(vputils.ByteStack, 0) // value stack

	// initialize module
	err := proc.SetPC(startAddress)
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
		pc1 := proc.PC()

		conditionals, err := proc.GetConditionals(mod.CodePage)
		if err != nil {
			message := err.Error() + " at PC " + pc1.ToString()
			return errors.New(message)
		}

		execute, err := conditionals.Evaluate(flags)
		if err != nil {
			return err
		}

		pc2 := proc.PC()

		opcode, err := proc.GetOpcode(mod.CodePage)
		if err != nil {
			message := err.Error() + " at PC " + pc2.ToString()
			return errors.New(message)
		}

		def := opcodeDefinitions[opcode]

		// get instruction definition (opcode and arguments)
		instruction, err := proc.DecodeInstruction(opcode, def, mod.CodePage, mod.DataPage)
		vputils.CheckAndExit(err)

		if trace {
			line := traceOpcode(pc1, opcode, def, flags, conditionals, instruction)
			fmt.Println(line)
		}

		// execute instruction
		syscall := byte(0)

		vStack, flags, syscall, err = proc.ExecuteOpcode(&mod.DataPage, opcode, vStack, instruction, execute, flags)

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
		line := traceHalt(proc.PC())
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

	proc := module.Processor{}
	err = executeCode(mod, proc, startAddress, trace, opcodeDefinitions)
	vputils.CheckAndExit(err)
}

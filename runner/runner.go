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

func executeCode(mod module.Module, proc module.Processor, startAddress vputils.Address, trace bool) error {
	// initialize virtual processor
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
	syscall := byte(0)

	for !halt {
		vStack, syscall, err = proc.ExecuteInstruction(vStack, mod.CodePage, &mod.DataPage, trace)
		if err != nil {
			return err
		}

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

	proc := module.Processor{}
	err = executeCode(mod, proc, startAddress, trace)
	vputils.CheckAndExit(err)
}

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

type Address struct {
	Bytes []byte
}

func (address Address) empty() bool {
	return len(address.Bytes) == 0
}

func (address Address) to_s() string {
	if len(address.Bytes) == 0 {
		return ""
	}

	value := 0
	for _, b := range address.Bytes {
		// should shift here
		// little-endian or big-endian?
		value += int(b)
	}

	return fmt.Sprintf("%04X", value)
}

func (address Address) ByteValue() byte {
	return address.Bytes[0]
}

func (ca Address) addByte(i int) Address {
	b := byte(i)
	a := ca.ByteValue() + b
	as := []byte{a}
	return Address{as}
}

type vector []byte

func (v vector) getByte(address Address) (byte, error) {
	max := len(v) - 1
	offset := int(address.ByteValue())
	if offset < 0 || offset > max {
		off := strconv.Itoa(offset)
		maxs := strconv.Itoa(max)
		return 0, errors.New("Index " + off + " out of range [0.." + maxs + "]")
	}

	return v[offset], nil
}

func (v vector) putByte(address Address, value byte) error {
	max := len(v) - 1
	offset := int(address.ByteValue())
	if offset < 0 || offset > max {
		off := strconv.Itoa(offset)
		maxs := strconv.Itoa(max)
		return errors.New("Index " + off + " out of range [0.." + maxs + "]")
	}

	v[offset] = value

	return nil
}

type Machine struct {
	BytesPerOpcode      int
	BytesPerCodeAddress int
	BytesPerDataAddress int
	Code                vector
	Data                vector
	Flags               []bool
}

func (machine Machine) getImmediateByte(pc Address) byte {
	codeAddress := pc.addByte(machine.BytesPerOpcode)

	value, err := machine.Code.getByte(codeAddress)
	vputils.CheckAndPanic(err)

	return value
}

func (machine Machine) getDirectAddress(pc Address) Address {
	codeAddress := pc.addByte(machine.BytesPerOpcode)

	dataAddr, err := machine.Code.getByte(codeAddress)
	vputils.CheckAndPanic(err)
	da := []byte{dataAddr}
	dataAddress := Address{da}

	return dataAddress
}

func (machine Machine) getDirectByte(pc Address) (byte, Address) {
	dataAddress := machine.getDirectAddress(pc)
	value, err := machine.Data.getByte(dataAddress)
	vputils.CheckAndPanic(err)

	return value, dataAddress
}

func (machine Machine) getIndirectAddress(pc Address) Address {
	dataAddress := machine.getDirectAddress(pc)
	dataAddr, err := machine.Data.getByte(dataAddress)
	vputils.CheckAndPanic(err)
	da := []byte{dataAddr}
	dataAddress = Address{da}

	return dataAddress
}

func (machine Machine) getIndirectByte(pc Address) (byte, Address) {
	dataAddress := machine.getIndirectAddress(pc)
	value, err := machine.Data.getByte(dataAddress)
	vputils.CheckAndPanic(err)

	return value, dataAddress
}

func (machine Machine) setFlags(value byte) {
	machine.Flags[0] = value == 0
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
	instructionDefinitions[0x08] = instructionDefinition{"OUT", "B", "S", ""}
	instructionDefinitions[0x11] = instructionDefinition{"FLAGS", "B", "D", ""}
	instructionDefinitions[0x12] = instructionDefinition{"FLAGS", "B", "I", ""}
	instructionDefinitions[0x13] = instructionDefinition{"FLAGS", "B", "S", ""}
	instructionDefinitions[0x21] = instructionDefinition{"INC", "B", "D", ""}
	instructionDefinitions[0x22] = instructionDefinition{"INC", "B", "I", ""}
	instructionDefinitions[0x90] = instructionDefinition{"JUMP", "", "", "A"}
	instructionDefinitions[0x92] = instructionDefinition{"JZ", "", "", "A"}

	return instructionDefinitions
}

func executeCode(code vector, startAddress int, data vector, trace bool, instructionDefinitions instructionTable) {
	bytesPerOpcode := 1
	bytesPerCodeAddress := 1
	bytesPerDataAddress := 1
	f := []bool{false}
	machine := Machine{bytesPerOpcode, bytesPerCodeAddress, bytesPerDataAddress, code, data, f}
	sa := byte(startAddress)
	pc := Address{[]byte{sa}}
	vStack := make(stack, 0)

	fmt.Printf("Execution started at %04x\n", pc.ByteValue())
	halt := false
	for !halt {
		opcode, err := code.getByte(pc)
		pcs := pc.to_s()
		vputils.CheckPrintAndExit(err, "at PC "+pcs)
		def := instructionDefinitions[opcode]
		value := byte(0)
		value_s := ""
		dataAddress := Address{[]byte{}}
		dataAddress1 := Address{[]byte{}}
		jumpAddress := Address{[]byte{}}

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

			value = machine.getImmediateByte(pc)
			value_s = fmt.Sprintf("%02X", value)
		}
		if def.AddressMode == "D" {
			instructionSize += bytesPerDataAddress

			dataAddress = machine.getDirectAddress(pc)

			value, _ = machine.getDirectByte(pc)
			value_s = fmt.Sprintf("%02X", value)
		}
		if def.AddressMode == "I" {
			instructionSize += bytesPerDataAddress

			dataAddress1 = machine.getDirectAddress(pc)
			dataAddress = machine.getIndirectAddress(pc)
			value, _ = machine.getIndirectByte(pc)
			value_s = fmt.Sprintf("%02X", value)
		}

		if def.JumpMode == "A" {
			instructionSize += bytesPerCodeAddress

			codeAddress := pc.addByte(bytesPerOpcode)
			jumpAddr, _ := code.getByte(codeAddress)
			jumpAddress = Address{[]byte{jumpAddr}}
		}
		if def.JumpMode == "R" {
			instructionSize += bytesPerCodeAddress

			codeAddress := pc.addByte(bytesPerOpcode)
			jumpAddr, _ := code.getByte(codeAddress)
			jumpAddress = Address{[]byte{jumpAddr}}
		}

		if trace {
			text := def.to_s()
			line := fmt.Sprintf("%s: %02X %s", pcs, opcode, text)
			if !dataAddress1.empty() {
				line += " @@" + dataAddress1.to_s()
			}
			if !dataAddress.empty() {
				line += " @" + dataAddress.to_s()
			}
			if len(value_s) > 0 {
				line += " =" + value_s
			}
			if !jumpAddress.empty() {
				line += " >" + jumpAddress.to_s()
			}
			fmt.Println(line)
		}

		switch opcode {
		case 0x00:
			// EXIT
			halt = true

			pc = pc.addByte(instructionSize)

		case 0x40:
			// PUSH.B immediate value
			vStack = vStack.push(value)

			pc = pc.addByte(instructionSize)

		case 0x41:
			// PUSH.B direct address
			vStack = vStack.push(value)

			pc = pc.addByte(instructionSize)

		case 0x42:
			// PUSH.B indirect address
			vStack = vStack.push(value)

			pc = pc.addByte(instructionSize)

		case 0x51:
			// POP.B direct address
			value, vStack, err = vStack.toppop()
			vputils.CheckAndPanic(err)

			err = data.putByte(dataAddress, value)
			vputils.CheckAndPanic(err)

			pc = pc.addByte(instructionSize)

		case 0x08:
			// OUT.B (implied stack)
			value, vStack, err = vStack.toppop()
			vputils.CheckAndPanic(err)

			fmt.Print(string(value))

			if trace {
				fmt.Println()
			}

			pc = pc.addByte(instructionSize)

		case 0x11:
			// FLAGS.B direct address
			machine.setFlags(value)

			pc = pc.addByte(instructionSize)

		case 0x12:
			// FLAGS.B indirect address
			machine.setFlags(value)

			pc = pc.addByte(instructionSize)

		case 0x13:
			// FLAGS.B (implied stack)
			value, err = vStack.top()
			vputils.CheckAndPanic(err)

			machine.setFlags(value)

			pc = pc.addByte(instructionSize)

		case 0x21:
			// INC.B direct address
			value += 1

			err = data.putByte(dataAddress, value)
			vputils.CheckAndPanic(err)

			pc = pc.addByte(instructionSize)

		case 0x22:
			// INC.B indirect address
			value += 1

			err = data.putByte(dataAddress, value)
			vputils.CheckAndPanic(err)

			pc = pc.addByte(instructionSize)

		case 0x90:
			// JUMP
			pc = jumpAddress

		case 0x92:
			// JZ
			if machine.Flags[0] {
				pc = jumpAddress
			} else {
				pc = pc.addByte(instructionSize)
			}

		default:
			// invalid opcode
			fmt.Printf("Invalid opcode %02x at %04x\n", opcode, pc.ByteValue())
			return
		}
	}

	fmt.Printf("Execution halted at %04x\n", pc.ByteValue())
}

func main() {
	tracePtr := flag.Bool("trace", false, "Display trace during execution.")
	flag.Parse()
	trace := *tracePtr
	args := flag.Args()

	if len(args) == 0 {
		fmt.Println("No module file specified")
		os.Exit(1)
	}

	moduleFile := args[0]
	fmt.Printf("Opening file '%s'\n", moduleFile)

	f, err := os.Open(moduleFile)
	vputils.CheckAndPanic(err)

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
		shortName := strings.Replace(nameValue.Name, " ", "", -1)
		if shortName == "CODEADDRESSWIDTH" {
			codeWidth = 1
		}
		if shortName == "DATAADDRESSWIDTH" {
			dataWidth = 1
		}
	}

	header = vputils.ReadString(f)
	if header != "exports" {
		fmt.Println("Did not find exports header")
		return
	}

	exports := vputils.ReadTextTable(f)

	startAddress := 0
	for _, nameValue := range exports {
		if nameValue.Name == "MAIN" {
			startAddress, err = strconv.Atoi(nameValue.Value)
			vputils.CheckAndExit(err)
		}
	}

	header = vputils.ReadString(f)
	if header != "code" {
		fmt.Println("Did not find code header")
		return
	}

	code := vputils.ReadBinaryBlock(f, codeWidth)

	header = vputils.ReadString(f)
	if header != "data" {
		fmt.Println("Did not find data header")
		return
	}

	data := vputils.ReadBinaryBlock(f, dataWidth)

	instructionDefinitions := defineInstructions()

	executeCode(code, startAddress, data, trace, instructionDefinitions)
}

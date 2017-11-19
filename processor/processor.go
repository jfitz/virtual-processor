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

func (address Address) to_s() string {
	value := 0
	for _, b := range address.Bytes {
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

func executeCode(code vector, startAddress int, data vector, trace bool, instructionDefinitions map[byte]string) {
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
		if trace {
			text := instructionDefinitions[opcode]
			fmt.Printf("%s: %02X %s\n", pcs, opcode, text)
		}

		instructionSize := 0
		switch opcode {
		case 0x00:
			// EXIT
			instructionSize = 1
			halt = true

			pc = pc.addByte(instructionSize)

		case 0x40:
			// PUSH.B immediate value
			instructionSize = bytesPerOpcode + 1

			value := machine.getImmediateByte(pc)
			vStack = vStack.push(value)

			pc = pc.addByte(instructionSize)

		case 0x41:
			// PUSH.B direct address
			instructionSize = bytesPerOpcode + bytesPerDataAddress

			value, _ := machine.getDirectByte(pc)
			vStack = vStack.push(value)

			pc = pc.addByte(instructionSize)

		case 0x42:
			// PUSH.B indirect address
			instructionSize = bytesPerOpcode + bytesPerDataAddress

			value, _ := machine.getIndirectByte(pc)
			vStack = vStack.push(value)

			pc = pc.addByte(instructionSize)

		case 0x51:
			// POP.B direct address
			instructionSize = bytesPerOpcode + bytesPerDataAddress

			value, vs, err := vStack.toppop()
			vputils.CheckAndPanic(err)
			vStack = vs

			dataAddress := machine.getDirectAddress(pc)
			err = data.putByte(dataAddress, value)
			vputils.CheckAndPanic(err)

			pc = pc.addByte(instructionSize)

		case 0x08:
			// OUT.B (implied stack)
			instructionSize = bytesPerOpcode

			c, vs, err := vStack.toppop()
			vputils.CheckAndPanic(err)
			vStack = vs

			fmt.Print(string(c))

			pc = pc.addByte(instructionSize)

		case 0x11:
			// FLAGS.B direct address
			instructionSize = bytesPerOpcode + bytesPerDataAddress

			value, _ := machine.getDirectByte(pc)
			machine.setFlags(value)

			pc = pc.addByte(instructionSize)

		case 0x12:
			// FLAGS.B indirect address
			instructionSize = bytesPerOpcode + bytesPerDataAddress

			value, _ := machine.getIndirectByte(pc)
			machine.setFlags(value)

			pc = pc.addByte(instructionSize)

		case 0x13:
			// FLAGS.B (implied stack)
			instructionSize = bytesPerOpcode

			value, err := vStack.top()
			vputils.CheckAndPanic(err)

			machine.setFlags(value)

			pc = pc.addByte(instructionSize)

		case 0x21:
			// INC.B direct address
			instructionSize = bytesPerOpcode + bytesPerDataAddress

			value, dataAddress := machine.getDirectByte(pc)
			value += 1

			err = data.putByte(dataAddress, value)
			vputils.CheckAndPanic(err)

			pc = pc.addByte(instructionSize)

		case 0x22:
			// INC.B indirect address
			instructionSize = bytesPerOpcode + bytesPerDataAddress

			value, dataAddress := machine.getIndirectByte(pc)
			value += 1

			err = data.putByte(dataAddress, value)
			vputils.CheckAndPanic(err)

			pc = pc.addByte(instructionSize)

		case 0x90:
			// JUMP
			instructionSize = bytesPerOpcode + bytesPerDataAddress

			codeAddress := pc.addByte(bytesPerOpcode)

			jumpAddr, err := code.getByte(codeAddress)
			vputils.CheckAndPanic(err)

			pc = Address{[]byte{jumpAddr}}

		case 0x92:
			// JZ
			instructionSize = bytesPerOpcode + bytesPerDataAddress

			codeAddress := pc.addByte(bytesPerOpcode)

			jumpAddr, err := code.getByte(codeAddress)
			vputils.CheckAndPanic(err)

			if machine.Flags[0] {
				pc = Address{[]byte{jumpAddr}}
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

func defineInstructions() map[byte]string {
	instructionDefinitions := make(map[byte]string)
	instructionDefinitions[0x00] = "EXIT"
	instructionDefinitions[0x40] = "PUSH.B immediate value"
	instructionDefinitions[0x41] = "PUSH.B direct address"
	instructionDefinitions[0x42] = "PUSH.B indirect address"
	instructionDefinitions[0x51] = "POP.B direct address"
	instructionDefinitions[0x08] = "OUT.B (implied stack)"
	instructionDefinitions[0x11] = "FLAGS.B direct address"
	instructionDefinitions[0x12] = "FLAGS.B indirect address"
	instructionDefinitions[0x13] = "FLAGS.B (implied stack)"
	instructionDefinitions[0x21] = "INC.B direct address"
	instructionDefinitions[0x22] = "INC.B indirect address"
	instructionDefinitions[0x90] = "JUMP"
	instructionDefinitions[0x92] = "JZ"

	return instructionDefinitions
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

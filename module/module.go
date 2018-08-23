/*
Package module for virtual-processor
*/
package module

import (
	"errors"
	"fmt"
	"github.com/jfitz/virtual-processor/vputils"
	"os"
	"strings"
)

// FlagsGroup --------------------
type FlagsGroup struct {
	Zero     bool
	Negative bool
	Positive bool
}

// ToString converts to string
func (flags FlagsGroup) ToString() string {
	s := ""

	if flags.Positive {
		s += " P"
	} else {
		s += " p"
	}

	if flags.Zero {
		s += " Z"
	} else {
		s += " z"
	}

	if flags.Negative {
		s += " N"
	} else {
		s += " n"
	}

	return s
}

// -------------------------------

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

// Module --------------------
type Module struct {
	Properties       []vputils.NameValue
	Code             vputils.Vector
	Exports          []vputils.NameValue
	Data             vputils.Vector
	CodeAddressWidth int
	DataAddressWidth int
	pc               vputils.Address
	RetStack         vputils.AddressStack
}

// Init - initialize
func (mod *Module) Init() {
}

// SetPC - set the PC
func (mod *Module) SetPC(address vputils.Address) error {
	if int(address.ByteValue()) >= len(mod.Code) {
		return errors.New("Address out of range")
	}

	mod.pc = address
	return nil
}

// PCByteValue - deprecated
func (mod Module) PCByteValue() byte {
	return mod.pc.ByteValue()
}

// PC - return current PC
func (mod Module) PC() vputils.Address {
	return mod.pc
}

// ImmediateByte - get a byte
func (mod Module) ImmediateByte() []byte {
	codeAddress := mod.pc.AddByte(1)

	value, err := mod.Code.GetByte(codeAddress)
	vputils.CheckAndExit(err)

	return []byte{value}
}

// ImmediateInt - get an I16
func (mod Module) ImmediateInt() []byte {
	codeAddress := mod.pc.AddByte(1)

	values := []byte{}

	value, err := mod.Code.GetByte(codeAddress)
	vputils.CheckAndExit(err)
	values = append(values, value)

	codeAddress = codeAddress.AddByte(1)

	value, err = mod.Code.GetByte(codeAddress)
	vputils.CheckAndExit(err)
	values = append(values, value)

	return values
}

// DirectAddress - get direct address
func (mod Module) DirectAddress() vputils.Address {
	codeAddress := mod.pc.AddByte(1)

	dataAddr, err := mod.Code.GetByte(codeAddress)
	vputils.CheckAndExit(err)
	da := []byte{dataAddr}
	dataAddress := vputils.Address{da, len(mod.Data)}

	return dataAddress
}

// DirectByte - get byte via direct address
func (mod Module) DirectByte() (byte, vputils.Address) {
	dataAddress := mod.DirectAddress()

	value, err := mod.Data.GetByte(dataAddress)
	vputils.CheckAndExit(err)

	return value, dataAddress
}

// IndirectAddress - get indirect address
func (mod Module) IndirectAddress() vputils.Address {
	dataAddress := mod.DirectAddress()
	dataAddr, err := mod.Data.GetByte(dataAddress)
	vputils.CheckAndExit(err)
	da := []byte{dataAddr}
	dataAddress = vputils.Address{da, len(mod.Data)}

	return dataAddress
}

// IndirectByte - get byte via indirect address
func (mod Module) IndirectByte() (byte, vputils.Address) {
	dataAddress := mod.IndirectAddress()
	value, err := mod.Data.GetByte(dataAddress)
	vputils.CheckAndExit(err)

	return value, dataAddress
}

// Push - push a value
func (mod *Module) Push(address vputils.Address) {
	mod.RetStack = mod.RetStack.Push(address)
}

// TopPop - pop the top value
func (mod *Module) TopPop() (vputils.Address, error) {
	address, retStack, err := mod.RetStack.TopPop()
	mod.RetStack = retStack

	return address, err
}

// ExecuteOpcode - execute one opcode
func ExecuteOpcode(mod *Module, opcode byte, vStack vputils.ByteStack, dataAddress vputils.Address, instructionSize int, jumpAddress vputils.Address, bytes []byte, execute bool, flags FlagsGroup, trace bool) (vputils.ByteStack, FlagsGroup, bool, error) {
	err := errors.New("")

	halt := false
	pc := mod.PC()
	newpc := pc

	bytes1 := []byte{}
	bytes2 := []byte{}

	// execute opcode
	switch opcode {
	case 0x00:
		// NOP
		newpc = pc.AddByte(instructionSize)

	case 0x04:
		// EXIT
		if execute {
			halt = true
		}

		// newpc = pc.AddByte(instructionSize)

	case 0x05:
		// KCALL - kernel call
		// update newpc before the call
		newpc = pc.AddByte(instructionSize)

		if execute {
			vStack = kernelCall(vStack)
		}

	case 0x08:
		// OUT (implied stack)
		if execute {
			bytes, vStack, err = vStack.PopByte(1)
			vputils.CheckAndPanic(err)

			fmt.Print(string(bytes[0]))

			if trace {
				fmt.Println()
			}
		}

		newpc = pc.AddByte(instructionSize)

	case 0x11:
		// FLAGS.B direct address
		if execute {
			flags.Zero = bytes[0] == 0
		}

		newpc = pc.AddByte(instructionSize)

	case 0x12:
		// FLAGS.B indirect address
		if execute {
			flags.Zero = bytes[0] == 0
		}

		newpc = pc.AddByte(instructionSize)

	case 0x13:
		// FLAGS.B (implied stack)
		if execute {
			bytes[0], err = vStack.TopByte()
			vputils.CheckAndPanic(err)

			flags.Zero = bytes[0] == 0
		}

		newpc = pc.AddByte(instructionSize)

	case 0x21:
		// INC.B direct address
		if execute {
			bytes[0]++

			err = mod.Data.PutByte(dataAddress, bytes[0])
			vputils.CheckAndPanic(err)
		}

		newpc = pc.AddByte(instructionSize)

	case 0x22:
		// INC.B indirect address
		if execute {
			bytes[0]++

			err = mod.Data.PutByte(dataAddress, bytes[0])
			vputils.CheckAndPanic(err)
		}

		newpc = pc.AddByte(instructionSize)

	case 0x31:
		// DEC.B direct address
		if execute {
			bytes[0]--

			err = mod.Data.PutByte(dataAddress, bytes[0])
			vputils.CheckAndPanic(err)
		}

		newpc = pc.AddByte(instructionSize)

	case 0x32:
		// DEC.B indirect address
		if execute {
			bytes[0]--

			err = mod.Data.PutByte(dataAddress, bytes[0])
			vputils.CheckAndPanic(err)
		}

		newpc = pc.AddByte(instructionSize)

	case 0x60:
		// PUSH.B immediate value
		if execute {
			vStack = vStack.PushBytes(bytes)
		}

		newpc = pc.AddByte(instructionSize)

	case 0x61:
		// PUSH.B direct address
		if execute {
			vStack = vStack.PushBytes(bytes)
		}

		newpc = pc.AddByte(instructionSize)

	case 0x62:
		// PUSH.B indirect address
		if execute {
			vStack = vStack.PushBytes(bytes)
		}

		newpc = pc.AddByte(instructionSize)

	case 0x64:
		// PUSH.I16 immediate value
		if execute {
			vStack = vStack.PushBytes(bytes)
		}

		newpc = pc.AddByte(instructionSize)

	case 0x65:
		// PUSH.I16 direct address
		if execute {
			vStack = vStack.PushBytes(bytes)
		}

		newpc = pc.AddByte(instructionSize)

	case 0x66:
		// PUSH.I16 indirect address
		if execute {
			vStack = vStack.PushBytes(bytes)
		}

		newpc = pc.AddByte(instructionSize)

	case 0x79:
		// PUSH.STR direct address
		if execute {
			s := ""
			address := dataAddress
			b := byte(1)

			for b != 0 {
				b, err = mod.Data.GetByte(address)
				vputils.CheckAndExit(err)
				c := string(b)
				s += c
				address = address.AddByte(1)
			}

			vStack = vStack.PushString(s)
		}

		newpc = pc.AddByte(instructionSize)

	case 0x81:
		// POP.B direct address
		if execute {
			bytes, vStack, err = vStack.PopByte(1)
			vputils.CheckAndExit(err)

			err = mod.Data.PutByte(dataAddress, bytes[0])
			vputils.CheckAndPanic(err)
		}

		newpc = pc.AddByte(instructionSize)

	case 0x83:
		// POP.B value (to nowhere)
		if execute {
			bytes, vStack, err = vStack.PopByte(1)
			vputils.CheckAndExit(err)
		}

		newpc = pc.AddByte(instructionSize)

	case 0xA0:
		// ADD.B
		if execute {
			bytes1, vStack, err = vStack.PopByte(1)
			vputils.CheckAndExit(err)

			bytes2, vStack, err = vStack.PopByte(1)
			vputils.CheckAndExit(err)

			value := bytes1[0] + bytes2[0]
			vStack = vStack.PushByte(value)
		}

		newpc = pc.AddByte(instructionSize)

	case 0xA1:
		// SUB.B
		if execute {
			bytes1, vStack, err = vStack.PopByte(1)
			vputils.CheckAndExit(err)

			bytes2, vStack, err = vStack.PopByte(1)
			vputils.CheckAndExit(err)

			value := bytes1[0] - bytes2[0]
			vStack = vStack.PushByte(value)
		}

		newpc = pc.AddByte(instructionSize)

	case 0xA2:
		// MUL.B
		if execute {
			bytes1, vStack, err = vStack.PopByte(1)
			vputils.CheckAndExit(err)

			bytes2, vStack, err = vStack.PopByte(1)
			vputils.CheckAndExit(err)

			value := bytes1[0] * bytes2[0]
			// TODO: push 2 bytes
			vStack = vStack.PushByte(value)
		}

		newpc = pc.AddByte(instructionSize)

	case 0xA3:
		// DIV.B
		if execute {
			bytes1, vStack, err = vStack.PopByte(1)
			vputils.CheckAndExit(err)

			bytes2, vStack, err = vStack.PopByte(1)
			vputils.CheckAndExit(err)

			value := bytes1[0] / bytes2[0]
			// TODO: push quotient and remainder (2 bytes)
			vStack = vStack.PushByte(value)
		}

		newpc = pc.AddByte(instructionSize)

	case 0xC0:
		// AND.B
		if execute {
			bytes1, vStack, err = vStack.PopByte(1)
			vputils.CheckAndExit(err)

			bytes2, vStack, err = vStack.PopByte(1)
			vputils.CheckAndExit(err)

			value := bytes1[0] & bytes2[0]
			vStack = vStack.PushByte(value)
		}

		newpc = pc.AddByte(instructionSize)

	case 0xC1:
		// OR.B
		if execute {
			bytes1, vStack, err = vStack.PopByte(1)
			vputils.CheckAndExit(err)

			bytes2, vStack, err = vStack.PopByte(1)
			vputils.CheckAndExit(err)

			value := bytes1[0] | bytes2[0]
			vStack = vStack.PushByte(value)
		}

		newpc = pc.AddByte(instructionSize)

	case 0xC3:
		// CMP.B
		if execute {
			bytes1, vStack, err = vStack.PopByte(1)
			vputils.CheckAndExit(err)

			bytes2, vStack, err = vStack.PopByte(1)
			vputils.CheckAndExit(err)

			value := bytes1[0] - bytes2[0]

			flags.Zero = value == 0
		}

		newpc = pc.AddByte(instructionSize)

	case 0xD0:
		// JUMP
		if execute {
			newpc = jumpAddress
		} else {
			newpc = pc.AddByte(instructionSize)
		}

	case 0xD1:
		// CALL
		if execute {
			newpc = jumpAddress
			retpc := pc.AddByte(instructionSize)
			mod.Push(retpc)
		} else {
			newpc = pc.AddByte(instructionSize)
		}

	case 0xD2:
		// RET
		if execute {
			newpc, err = mod.TopPop()
			vputils.CheckAndExit(err)
		} else {
			newpc = pc.AddByte(instructionSize)
		}

	default:
		// invalid opcode
		s := fmt.Sprintf("Invalid opcode %02x at %s\n", opcode, pc.ToString())
		return vStack, flags, halt, errors.New(s)
	}

	// advance to next instruction
	err = mod.SetPC(newpc)
	if err != nil {
		s := fmt.Sprintf("Invalid address %s for PC in main: %s", newpc.ToString(), err.Error())
		err = errors.New(s)
	}

	return vStack, flags, halt, err
}

// Write a module to a file
func (mod Module) Write(filename string) {
	f, err := os.Create(filename)
	vputils.CheckAndPanic(err)

	defer f.Close()

	vputils.WriteString(f, "module")

	vputils.WriteTextTable("properties", mod.Properties, f)
	vputils.WriteTextTable("exports", mod.Exports, f)
	vputils.WriteBinaryBlock("code", mod.Code, f, mod.CodeAddressWidth)
	vputils.WriteBinaryBlock("data", mod.Data, f, mod.DataAddressWidth)

	f.Sync()
}

// Read a file into a module
func Read(moduleFile string) (Module, error) {
	f, err := os.Open(moduleFile)
	vputils.CheckAndExit(err)

	defer f.Close()

	header := vputils.ReadString(f)
	if header != "module" {
		return Module{}, errors.New("Did not find module header")
	}

	header = vputils.ReadString(f)
	if header != "properties" {
		return Module{}, errors.New("Did not find properties header")
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
		return Module{}, errors.New("Did not find exports header")
	}

	exports := vputils.ReadTextTable(f)

	header = vputils.ReadString(f)
	if header != "code" {
		return Module{}, errors.New("Did not find code header")
	}

	code := vputils.ReadBinaryBlock(f, codeAddressWidth)

	header = vputils.ReadString(f)
	if header != "data" {
		return Module{}, errors.New("Did not find data header")
	}

	data := vputils.ReadBinaryBlock(f, dataAddressWidth)

	mod := Module{
		Properties:       properties,
		Code:             code,
		Exports:          exports,
		Data:             data,
		CodeAddressWidth: codeAddressWidth,
		DataAddressWidth: dataAddressWidth,
	}

	mod.Init()

	return mod, nil
}

// -------------------------------

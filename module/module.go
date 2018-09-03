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

func decodeConditional(conditional byte) string {
	condiString := ""

	switch conditional {
	case 0xE0:
		condiString = "ZERO"
	case 0xE8:
		condiString = "NOT"
	default:
		condiString = "ERROR"
	}

	return condiString
}

// Conditionals for modifiers on opcodes
type Conditionals []byte

// ToString - convert to string
func (conditionals Conditionals) ToString() string {
	ss := []string{}

	for _, conditional := range conditionals {
		s := decodeConditional(conditional)
		ss = append(ss, s)
	}

	result := strings.Join(ss, " ")

	return result
}

// ToByteString - convert to string of byte representations
func (conditionals Conditionals) ToByteString() string {
	return fmt.Sprintf("%02X ", conditionals)
}

// Evaluate - evaluate as true or false
func (conditionals Conditionals) Evaluate(flags FlagsGroup) (bool, error) {
	execute := true
	stack := make(vputils.BoolStack, 0)

	for _, conditional := range conditionals {
		switch conditional {
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

// -------------------------------

// InstructionDefinition ---------
type InstructionDefinition struct {
	FullOpcode  []byte
	Address1    vputils.Address
	Address     vputils.Address
	Size        int
	JumpAddress vputils.Address
	Bytes       []byte
	ValueStr    string
}

// ToByteString - convert full opcode to printable
func (def InstructionDefinition) ToByteString() string {
	s := ""

	for _, b := range def.FullOpcode {
		s += fmt.Sprintf("%02X ", b)
	}

	return s
}

// Page --------------------------
type Page struct {
	Properties []vputils.NameValue
	Contents   vputils.Vector
}

// Module ------------------------
type Module struct {
	Properties       []vputils.NameValue
	CodePage         Page
	Exports          []vputils.NameValue
	DataPage         Page
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
	if int(address.ByteValue()) >= len(mod.CodePage.Contents) {
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

// IncPC - increment the PC
func (mod *Module) IncPC() {
	mod.pc = mod.pc.AddByte(1)
}

// ImmediateByte - get a byte
func (mod Module) ImmediateByte() ([]byte, error) {
	codeAddress := mod.pc.AddByte(1)

	value, err := mod.CodePage.Contents.GetByte(codeAddress)
	if err != nil {
		return []byte{}, err
	}

	return []byte{value}, nil
}

// ImmediateInt - get an I16
func (mod Module) ImmediateInt() ([]byte, error) {
	codeAddress := mod.pc.AddByte(1)

	values := []byte{}

	value, err := mod.CodePage.Contents.GetByte(codeAddress)
	if err != nil {
		return []byte{}, err
	}

	values = append(values, value)

	codeAddress = codeAddress.AddByte(1)

	value, err = mod.CodePage.Contents.GetByte(codeAddress)
	if err != nil {
		return []byte{}, err
	}

	values = append(values, value)

	return values, nil
}

// DirectAddress - get direct address
func (mod Module) DirectAddress() (vputils.Address, error) {
	emptyAddress, _ := vputils.MakeAddress(0, 1, 0)

	codeAddress := mod.pc.AddByte(1)

	dataAddr, err := mod.CodePage.Contents.GetByte(codeAddress)
	if err != nil {
		return emptyAddress, err
	}

	da := []byte{dataAddr}
	dataAddress := vputils.Address{da, len(mod.DataPage.Contents)}

	return dataAddress, nil
}

// DirectByte - get byte via direct address
func (mod Module) DirectByte() (byte, error) {
	dataAddress, err := mod.DirectAddress()
	if err != nil {
		return 0, err
	}

	value, err := mod.DataPage.Contents.GetByte(dataAddress)
	if err != nil {
		return 0, err
	}

	return value, nil
}

// IndirectAddress - get indirect address
func (mod Module) IndirectAddress() (vputils.Address, error) {
	emptyAddress, _ := vputils.MakeAddress(0, 1, 0)

	dataAddress, err := mod.DirectAddress()
	if err != nil {
		return emptyAddress, err
	}

	dataAddr, err := mod.DataPage.Contents.GetByte(dataAddress)
	if err != nil {
		return emptyAddress, err
	}

	da := []byte{dataAddr}
	dataAddress = vputils.Address{da, len(mod.DataPage.Contents)}

	return dataAddress, nil
}

// IndirectByte - get byte via indirect address
func (mod Module) IndirectByte() (byte, error) {
	dataAddress, err := mod.IndirectAddress()
	if err != nil {
		return 0, err
	}

	value, err := mod.DataPage.Contents.GetByte(dataAddress)
	if err != nil {
		return 0, err
	}

	return value, nil
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

// GetConditionals - get the conditionals for instruction at PC
func (mod *Module) GetConditionals() (Conditionals, error) {
	conditionals := Conditionals{}
	err := errors.New("")

	myByte, err := mod.CodePage.Contents.GetByte(mod.pc)

	hasConditional := true

	for hasConditional {
		if myByte >= 0xE0 && myByte <= 0xEF {
			conditionals = append(conditionals, myByte)
			mod.IncPC()
			myByte, err = mod.CodePage.Contents.GetByte(mod.pc)
		} else {
			hasConditional = false
		}
	}

	return conditionals, err
}

// GetOpcode - get the opcode at PC
func (mod Module) GetOpcode() (byte, error) {
	return mod.CodePage.Contents.GetByte(mod.pc)
}

// ExecuteOpcode - execute one opcode
func (mod *Module) ExecuteOpcode(opcode byte, vStack vputils.ByteStack, instruction InstructionDefinition, execute bool, flags FlagsGroup) (vputils.ByteStack, FlagsGroup, byte, error) {
	dataAddress := instruction.Address
	instructionSize := instruction.Size
	jumpAddress := instruction.JumpAddress
	bytes := instruction.Bytes

	err := errors.New("")

	syscall := byte(0)
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
			syscall = opcode
		}

		// newpc = pc.AddByte(instructionSize)

	case 0x05:
		// KCALL - kernel call
		if execute {
			syscall = opcode
		}

		newpc = pc.AddByte(instructionSize)

	case 0x08:
		// OUT (implied stack)
		if execute {
			syscall = opcode
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
			buffer, err := vStack.TopByte()
			vputils.CheckAndPanic(err)

			flags.Zero = buffer == 0
		}

		newpc = pc.AddByte(instructionSize)

	case 0x21:
		// INC.B direct address
		if execute {
			bytes[0]++

			err = mod.DataPage.Contents.PutByte(dataAddress, bytes[0])
			vputils.CheckAndPanic(err)
		}

		newpc = pc.AddByte(instructionSize)

	case 0x22:
		// INC.B indirect address
		if execute {
			bytes[0]++

			err = mod.DataPage.Contents.PutByte(dataAddress, bytes[0])
			vputils.CheckAndPanic(err)
		}

		newpc = pc.AddByte(instructionSize)

	case 0x31:
		// DEC.B direct address
		if execute {
			bytes[0]--

			err = mod.DataPage.Contents.PutByte(dataAddress, bytes[0])
			vputils.CheckAndPanic(err)
		}

		newpc = pc.AddByte(instructionSize)

	case 0x32:
		// DEC.B indirect address
		if execute {
			bytes[0]--

			err = mod.DataPage.Contents.PutByte(dataAddress, bytes[0])
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
				b, err = mod.DataPage.Contents.GetByte(address)
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

			err = mod.DataPage.Contents.PutByte(dataAddress, bytes[0])
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
		return vStack, flags, 0, errors.New(s)
	}

	// advance to next instruction
	err = mod.SetPC(newpc)
	if err != nil {
		s := fmt.Sprintf("Invalid address %s for PC in main: %s", newpc.ToString(), err.Error())
		err = errors.New(s)
	}

	return vStack, flags, syscall, err
}

// Write a module to a file
func (mod Module) Write(filename string) {
	f, err := os.Create(filename)
	vputils.CheckAndPanic(err)

	defer f.Close()

	vputils.WriteString(f, "module")

	vputils.WriteTextTable("properties", mod.Properties, f)
	vputils.WriteTextTable("exports", mod.Exports, f)
	vputils.WriteTextTable("code_properties", mod.CodePage.Properties, f)
	vputils.WriteBinaryBlock("code", mod.CodePage.Contents, f, mod.CodeAddressWidth)
	vputils.WriteTextTable("data_properties", mod.DataPage.Properties, f)
	vputils.WriteBinaryBlock("data", mod.DataPage.Contents, f, mod.DataAddressWidth)

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

	properties, err := vputils.ReadTextTable(f)
	vputils.CheckAndExit(err)

	codeAddressWidth := 1
	dataAddressWidth := 1

	header = vputils.ReadString(f)
	if header != "exports" {
		return Module{}, errors.New("Did not find exports header")
	}

	exports, err := vputils.ReadTextTable(f)
	vputils.CheckAndExit(err)

	header = vputils.ReadString(f)
	if header != "code_properties" {
		return Module{}, errors.New("Did not find code_properties header")
	}

	codeProperties, err := vputils.ReadTextTable(f)
	vputils.CheckAndExit(err)

	header = vputils.ReadString(f)
	if header != "code" {
		return Module{}, errors.New("Did not find code header")
	}

	code, err := vputils.ReadBinaryBlock(f, codeAddressWidth)
	vputils.CheckAndExit(err)

	codePage := Page{codeProperties, code}

	header = vputils.ReadString(f)
	if header != "data_properties" {
		return Module{}, errors.New("Did not find data_properties header")
	}

	dataProperties, err := vputils.ReadTextTable(f)
	vputils.CheckAndExit(err)

	header = vputils.ReadString(f)
	if header != "data" {
		return Module{}, errors.New("Did not find data header")
	}

	data, err := vputils.ReadBinaryBlock(f, dataAddressWidth)
	vputils.CheckAndExit(err)

	dataPage := Page{dataProperties, data}

	// TODO: check data page datawidth is the same as code page datawidth

	mod := Module{
		Properties:       properties,
		CodePage:         codePage,
		Exports:          exports,
		DataPage:         dataPage,
		CodeAddressWidth: codeAddressWidth,
		DataAddressWidth: dataAddressWidth,
	}

	mod.Init()

	return mod, nil
}

// -------------------------------

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
// -------------------------------

// OpcodeDefinition --------------
type OpcodeDefinition struct {
	Name        string
	Width       string
	AddressMode string
}

// ToString ----------------------
func (def OpcodeDefinition) ToString() string {
	s := def.Name

	if len(def.Width) > 0 {
		s += " "
		s += def.Width
	}

	return s
}

// OpcodeSize --------------------
func (def OpcodeDefinition) OpcodeSize() int {
	return 1
}

// TargetSize --------------------
func (def OpcodeDefinition) TargetSize() int {
	targetSize := 0

	if def.Width == "BYTE" {
		targetSize = 1
	}
	if def.Width == "I16" {
		targetSize = 2
	}
	if def.Width == "I32" {
		targetSize = 4
	}
	if def.Width == "I64" {
		targetSize = 8
	}
	if def.Width == "F32" {
		targetSize = 4
	}
	if def.Width == "F64" {
		targetSize = 8
	}

	return targetSize
}

// -------------------------------
// -------------------------------

// OpcodeTable --------
type OpcodeTable map[byte]OpcodeDefinition

// DefineOpcodes - define the table of opcodes
func DefineOpcodes() OpcodeTable {
	opcodeDefinitions := make(OpcodeTable)

	opcodeDefinitions[0x00] = OpcodeDefinition{"NOP", "", ""}
	opcodeDefinitions[0x04] = OpcodeDefinition{"EXIT", "", ""}
	opcodeDefinitions[0x05] = OpcodeDefinition{"KCALL", "", ""}
	opcodeDefinitions[0x08] = OpcodeDefinition{"OUT", "", "S"}

	opcodeDefinitions[0x60] = OpcodeDefinition{"PUSH", "BYTE", "V"}
	opcodeDefinitions[0x61] = OpcodeDefinition{"PUSH", "BYTE", "D"}
	opcodeDefinitions[0x62] = OpcodeDefinition{"PUSH", "BYTE", "I"}

	opcodeDefinitions[0x64] = OpcodeDefinition{"PUSH", "I16", "V"}
	opcodeDefinitions[0x65] = OpcodeDefinition{"PUSH", "I16", "D"}
	opcodeDefinitions[0x66] = OpcodeDefinition{"PUSH", "I16", "I"}

	opcodeDefinitions[0x79] = OpcodeDefinition{"PUSH", "STRING", "D"}

	opcodeDefinitions[0x81] = OpcodeDefinition{"POP", "BYTE", "D"}
	opcodeDefinitions[0x82] = OpcodeDefinition{"POP", "BYTE", "I"}
	opcodeDefinitions[0x83] = OpcodeDefinition{"POP", "BYTE", "S"}

	opcodeDefinitions[0x11] = OpcodeDefinition{"FLAGS", "BYTE", "D"}
	opcodeDefinitions[0x12] = OpcodeDefinition{"FLAGS", "BYTE", "I"}
	opcodeDefinitions[0x13] = OpcodeDefinition{"FLAGS", "BYTE", "S"}

	opcodeDefinitions[0x21] = OpcodeDefinition{"INC", "BYTE", "D"}
	opcodeDefinitions[0x22] = OpcodeDefinition{"INC", "BYTE", "I"}
	opcodeDefinitions[0x31] = OpcodeDefinition{"DEC", "BYTE", "D"}
	opcodeDefinitions[0x32] = OpcodeDefinition{"DEC", "BYTE", "I"}

	opcodeDefinitions[0xD0] = OpcodeDefinition{"JUMP", "", ""}
	opcodeDefinitions[0xD1] = OpcodeDefinition{"CALL", "", ""}
	opcodeDefinitions[0xD2] = OpcodeDefinition{"RET", "", ""}

	opcodeDefinitions[0xA0] = OpcodeDefinition{"ADD", "BYTE", ""}
	opcodeDefinitions[0xA1] = OpcodeDefinition{"SUB", "BYTE", ""}
	opcodeDefinitions[0xA2] = OpcodeDefinition{"MUL", "BYTE", ""}
	opcodeDefinitions[0xA3] = OpcodeDefinition{"DIV", "BYTE", ""}

	opcodeDefinitions[0xC0] = OpcodeDefinition{"AND", "BYTE", ""}
	opcodeDefinitions[0xC1] = OpcodeDefinition{"OR", "BYTE", ""}
	opcodeDefinitions[0xC3] = OpcodeDefinition{"CMP", "BYTE", ""}

	return opcodeDefinitions
}

// --------------------
// --------------------

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

// Processor ---------------------
type Processor struct {
	pc       vputils.Address
	RetStack vputils.AddressStack
}

// SetPC - set the PC
func (proc *Processor) SetPC(address vputils.Address) error {
	proc.pc = address
	return nil
}

// PC - return current PC
func (proc Processor) PC() vputils.Address {
	return proc.pc
}

// IncPC - increment the PC
func (proc *Processor) IncPC() {
	proc.pc = proc.pc.Increment(1)
}

// Push - push a value
func (proc *Processor) Push(address vputils.Address) {
	proc.RetStack = proc.RetStack.Push(address)
}

// TopPop - pop the top value
func (proc *Processor) TopPop() (vputils.Address, error) {
	address, retStack, err := proc.RetStack.TopPop()
	proc.RetStack = retStack

	return address, err
}

// GetConditionals - get the conditionals for instruction at PC
func (proc *Processor) GetConditionals(code Page) (Conditionals, error) {
	conditionals := Conditionals{}
	err := errors.New("")

	myByte, err := code.Contents.GetByte(proc.PC())
	if err != nil {
		return conditionals, err
	}

	hasConditional := true

	for hasConditional {
		if myByte >= 0xE0 && myByte <= 0xEF {
			conditionals = append(conditionals, myByte)
			// step PC over conditionals
			proc.IncPC()
			myByte, err = code.Contents.GetByte(proc.PC())
			if err != nil {
				return conditionals, err
			}
		} else {
			hasConditional = false
		}
	}

	return conditionals, nil
}

// GetOpcode - get the opcode at PC
func (proc Processor) GetOpcode(code Page) (byte, error) {
	return code.Contents.GetByte(proc.PC())
}

// ImmediateByte - get a byte
func (proc Processor) ImmediateByte(code Page) ([]byte, error) {
	pc := proc.PC()
	codeAddress := pc.Increment(1)

	value, err := code.Contents.GetByte(codeAddress)
	if err != nil {
		return []byte{}, err
	}

	return []byte{value}, nil
}

// ImmediateInt - get an I16
func (proc Processor) ImmediateInt(code Page) ([]byte, error) {
	pc := proc.PC()
	codeAddress := pc.Increment(1)

	values := []byte{}

	value, err := code.Contents.GetByte(codeAddress)
	if err != nil {
		return []byte{}, err
	}

	values = append(values, value)

	codeAddress = codeAddress.Increment(1)

	value, err = code.Contents.GetByte(codeAddress)
	if err != nil {
		return []byte{}, err
	}

	values = append(values, value)

	return values, nil
}

// JumpAddress - get direct address
func (proc Processor) JumpAddress(code Page) (vputils.Address, error) {
	emptyAddress, _ := vputils.MakeAddress(0, 1, 0)

	pc := proc.PC()
	codeAddress := pc.Increment(1)

	jumpAddr, err := code.Contents.GetByte(codeAddress)
	if err != nil {
		return emptyAddress, err
	}

	jumpAddress := vputils.Address{int(jumpAddr), 1, len(code.Contents)}

	return jumpAddress, nil
}

// DirectAddress - get direct address
func (proc Processor) DirectAddress(code Page, data Page) (vputils.Address, error) {
	emptyAddress, _ := vputils.MakeAddress(0, 1, 0)

	pc := proc.PC()
	codeAddress := pc.Increment(1)

	dataAddr, err := code.Contents.GetByte(codeAddress)
	if err != nil {
		return emptyAddress, err
	}

	dataAddress := vputils.Address{int(dataAddr), 1, len(data.Contents)}

	return dataAddress, nil
}

// DirectByte - get byte via direct address
func (proc Processor) DirectByte(code Page, data Page) (byte, error) {
	dataAddress, err := proc.DirectAddress(code, data)
	if err != nil {
		return 0, err
	}

	value, err := data.Contents.GetByte(dataAddress)
	if err != nil {
		return 0, err
	}

	return value, nil
}

// IndirectAddress - get indirect address
func (proc Processor) IndirectAddress(code Page, data Page) (vputils.Address, error) {
	emptyAddress, _ := vputils.MakeAddress(0, 1, 0)

	dataAddress, err := proc.DirectAddress(code, data)
	if err != nil {
		return emptyAddress, err
	}

	dataAddr, err := data.Contents.GetByte(dataAddress)
	if err != nil {
		return emptyAddress, err
	}

	dataAddress = vputils.Address{int(dataAddr), 1, len(data.Contents)}

	return dataAddress, nil
}

// IndirectByte - get byte via indirect address
func (proc Processor) IndirectByte(code Page, data Page) (byte, error) {
	dataAddress, err := proc.IndirectAddress(code, data)
	if err != nil {
		return 0, err
	}

	value, err := data.Contents.GetByte(dataAddress)
	if err != nil {
		return 0, err
	}

	return value, nil
}

func (proc Processor) DecodeInstruction(opcode byte, def OpcodeDefinition, code Page, data Page) (InstructionDefinition, error) {
	fullOpcode := []byte{opcode}

	// working bytes for opcode
	workBytes := []byte{}

	// addresses for opcode
	dataAddress := vputils.Address{0, 0, 0}
	dataAddress1 := vputils.Address{0, 0, 0}
	jumpAddress := vputils.Address{0, 0, 0}
	valueStr := ""

	instructionSize := def.OpcodeSize()
	targetSize := def.TargetSize()

	err := errors.New("")

	// decode immediate value
	if def.AddressMode == "V" {

		switch def.Width {

		case "BYTE":
			workBytes, err = proc.ImmediateByte(code)
			if err != nil {
				return InstructionDefinition{}, err
			}

			valueStr = fmt.Sprintf("%02X", workBytes[0])

		case "I16":
			workBytes, err = proc.ImmediateInt(code)
			if err != nil {
				return InstructionDefinition{}, err
			}

			valueStr = fmt.Sprintf("%02X%02X", workBytes[1], workBytes[0])

		}

		fullOpcode = append(fullOpcode, workBytes...)
		instructionSize += targetSize
	}

	// decode memory target
	if def.AddressMode == "D" {
		dataAddress, err = proc.DirectAddress(code, data)
		if err != nil {
			return InstructionDefinition{}, err
		}

		bytes := dataAddress.ToBytes()
		fullOpcode = append(fullOpcode, bytes...)

		buffer, err := proc.DirectByte(code, data)
		if err != nil {
			return InstructionDefinition{}, err
		}

		workBytes = append(workBytes, buffer)
		valueStr = fmt.Sprintf("%02X", buffer)

		instructionSize += dataAddress.Size
	}

	if def.AddressMode == "I" {
		dataAddress1, err = proc.DirectAddress(code, data)
		if err != nil {
			return InstructionDefinition{}, err
		}

		bytes1 := dataAddress1.ToBytes()
		fullOpcode = append(fullOpcode, bytes1...)
		bytes := dataAddress.ToBytes()
		workBytes = append(workBytes, bytes...)

		dataAddress, err = proc.IndirectAddress(code, data)
		if err != nil {
			return InstructionDefinition{}, err
		}

		buffer, err := proc.IndirectByte(code, data)
		if err != nil {
			return InstructionDefinition{}, err
		}

		workBytes = append(workBytes, buffer)
		valueStr = fmt.Sprintf("%02X", buffer)

		instructionSize += dataAddress1.Size
	}

	// decode jump/call target
	if opcode == 0xD0 || opcode == 0xD1 {
		jumpAddress, err = proc.JumpAddress(code)
		if err != nil {
			return InstructionDefinition{}, err
		}

		bytes := jumpAddress.ToBytes()
		fullOpcode = append(fullOpcode, bytes...)
		instructionSize += jumpAddress.Size
	}

	instruction := InstructionDefinition{fullOpcode, dataAddress1, dataAddress, instructionSize, jumpAddress, workBytes, valueStr}

	return instruction, nil
}

// ExecuteOpcode - execute one opcode
func (proc *Processor) ExecuteOpcode(data *Page, opcode byte, vStack vputils.ByteStack, instruction InstructionDefinition, execute bool, flags FlagsGroup) (vputils.ByteStack, FlagsGroup, byte, error) {
	dataAddress := instruction.Address
	instructionSize := instruction.Size
	jumpAddress := instruction.JumpAddress
	bytes := instruction.Bytes

	err := errors.New("")

	syscall := byte(0)

	pc := proc.PC()
	newpc := pc

	bytes1 := []byte{}
	bytes2 := []byte{}

	// execute opcode
	switch opcode {
	case 0x00:
		// NOP
		newpc = pc.Increment(instructionSize)

	case 0x04:
		// EXIT
		if execute {
			syscall = opcode
		}

		// newpc = pc.Increment(instructionSize)

	case 0x05:
		// KCALL - kernel call
		if execute {
			syscall = opcode
		}

		newpc = pc.Increment(instructionSize)

	case 0x08:
		// OUT (implied stack)
		if execute {
			syscall = opcode
		}

		newpc = pc.Increment(instructionSize)

	case 0x11:
		// FLAGS.B direct address
		if execute {
			flags.Zero = bytes[0] == 0
		}

		newpc = pc.Increment(instructionSize)

	case 0x12:
		// FLAGS.B indirect address
		if execute {
			flags.Zero = bytes[0] == 0
		}

		newpc = pc.Increment(instructionSize)

	case 0x13:
		// FLAGS.B (implied stack)
		if execute {
			buffer, err := vStack.TopByte()
			if err != nil {
				return vStack, flags, syscall, err
			}

			flags.Zero = buffer == 0
		}

		newpc = pc.Increment(instructionSize)

	case 0x21:
		// INC.B direct address
		if execute {
			bytes[0]++

			err = data.Contents.PutByte(dataAddress, bytes[0])
			if err != nil {
				return vStack, flags, syscall, err
			}
		}

		newpc = pc.Increment(instructionSize)

	case 0x22:
		// INC.B indirect address
		if execute {
			bytes[0]++

			err = data.Contents.PutByte(dataAddress, bytes[0])
			if err != nil {
				return vStack, flags, syscall, err
			}
		}

		newpc = pc.Increment(instructionSize)

	case 0x31:
		// DEC.B direct address
		if execute {
			bytes[0]--

			err = data.Contents.PutByte(dataAddress, bytes[0])
			if err != nil {
				return vStack, flags, syscall, err
			}
		}

		newpc = pc.Increment(instructionSize)

	case 0x32:
		// DEC.B indirect address
		if execute {
			bytes[0]--

			err = data.Contents.PutByte(dataAddress, bytes[0])
			if err != nil {
				return vStack, flags, syscall, err
			}
		}

		newpc = pc.Increment(instructionSize)

	case 0x60:
		// PUSH.B immediate value
		if execute {
			vStack = vStack.PushBytes(bytes)
		}

		newpc = pc.Increment(instructionSize)

	case 0x61:
		// PUSH.B direct address
		if execute {
			vStack = vStack.PushBytes(bytes)
		}

		newpc = pc.Increment(instructionSize)

	case 0x62:
		// PUSH.B indirect address
		if execute {
			vStack = vStack.PushBytes(bytes)
		}

		newpc = pc.Increment(instructionSize)

	case 0x64:
		// PUSH.I16 immediate value
		if execute {
			vStack = vStack.PushBytes(bytes)
		}

		newpc = pc.Increment(instructionSize)

	case 0x65:
		// PUSH.I16 direct address
		if execute {
			vStack = vStack.PushBytes(bytes)
		}

		newpc = pc.Increment(instructionSize)

	case 0x66:
		// PUSH.I16 indirect address
		if execute {
			vStack = vStack.PushBytes(bytes)
		}

		newpc = pc.Increment(instructionSize)

	case 0x79:
		// PUSH.STR direct address
		if execute {
			s := ""
			address := dataAddress
			b := byte(1)

			for b != 0 {
				b, err = data.Contents.GetByte(address)
				if err != nil {
					return vStack, flags, syscall, err
				}

				c := string(b)
				s += c
				address = address.Increment(1)
			}

			vStack = vStack.PushString(s)
		}

		newpc = pc.Increment(instructionSize)

	case 0x81:
		// POP.B direct address
		if execute {
			bytes, vStack, err = vStack.PopByte(1)
			if err != nil {
				return vStack, flags, syscall, err
			}

			err = data.Contents.PutByte(dataAddress, bytes[0])
			if err != nil {
				return vStack, flags, syscall, err
			}
		}

		newpc = pc.Increment(instructionSize)

	case 0x83:
		// POP.B value (to nowhere)
		if execute {
			bytes, vStack, err = vStack.PopByte(1)
			if err != nil {
				return vStack, flags, syscall, err
			}
		}

		newpc = pc.Increment(instructionSize)

	case 0xA0:
		// ADD.B
		if execute {
			bytes1, vStack, err = vStack.PopByte(1)
			if err != nil {
				return vStack, flags, syscall, err
			}

			bytes2, vStack, err = vStack.PopByte(1)
			if err != nil {
				return vStack, flags, syscall, err
			}

			value := bytes1[0] + bytes2[0]
			vStack = vStack.PushByte(value)
		}

		newpc = pc.Increment(instructionSize)

	case 0xA1:
		// SUB.B
		if execute {
			bytes1, vStack, err = vStack.PopByte(1)
			if err != nil {
				return vStack, flags, syscall, err
			}

			bytes2, vStack, err = vStack.PopByte(1)
			if err != nil {
				return vStack, flags, syscall, err
			}

			value := bytes1[0] - bytes2[0]
			vStack = vStack.PushByte(value)
		}

		newpc = pc.Increment(instructionSize)

	case 0xA2:
		// MUL.B
		if execute {
			bytes1, vStack, err = vStack.PopByte(1)
			if err != nil {
				return vStack, flags, syscall, err
			}

			bytes2, vStack, err = vStack.PopByte(1)
			if err != nil {
				return vStack, flags, syscall, err
			}

			value := bytes1[0] * bytes2[0]
			// TODO: push 2 bytes
			vStack = vStack.PushByte(value)
		}

		newpc = pc.Increment(instructionSize)

	case 0xA3:
		// DIV.B
		if execute {
			bytes1, vStack, err = vStack.PopByte(1)
			if err != nil {
				return vStack, flags, syscall, err
			}

			bytes2, vStack, err = vStack.PopByte(1)
			if err != nil {
				return vStack, flags, syscall, err
			}

			value := bytes1[0] / bytes2[0]
			// TODO: push quotient and remainder (2 bytes)
			vStack = vStack.PushByte(value)
		}

		newpc = pc.Increment(instructionSize)

	case 0xC0:
		// AND.B
		if execute {
			bytes1, vStack, err = vStack.PopByte(1)
			if err != nil {
				return vStack, flags, syscall, err
			}

			bytes2, vStack, err = vStack.PopByte(1)
			if err != nil {
				return vStack, flags, syscall, err
			}

			value := bytes1[0] & bytes2[0]
			vStack = vStack.PushByte(value)
		}

		newpc = pc.Increment(instructionSize)

	case 0xC1:
		// OR.B
		if execute {
			bytes1, vStack, err = vStack.PopByte(1)
			if err != nil {
				return vStack, flags, syscall, err
			}

			bytes2, vStack, err = vStack.PopByte(1)
			if err != nil {
				return vStack, flags, syscall, err
			}

			value := bytes1[0] | bytes2[0]
			vStack = vStack.PushByte(value)
		}

		newpc = pc.Increment(instructionSize)

	case 0xC3:
		// CMP.B
		if execute {
			bytes1, vStack, err = vStack.PopByte(1)
			if err != nil {
				return vStack, flags, syscall, err
			}

			bytes2, vStack, err = vStack.PopByte(1)
			if err != nil {
				return vStack, flags, syscall, err
			}

			value := bytes1[0] - bytes2[0]

			flags.Zero = value == 0
		}

		newpc = pc.Increment(instructionSize)

	case 0xD0:
		// JUMP
		if execute {
			newpc = jumpAddress
		} else {
			newpc = pc.Increment(instructionSize)
		}

	case 0xD1:
		// CALL
		if execute {
			newpc = jumpAddress
			retpc := pc.Increment(instructionSize)
			proc.Push(retpc)
		} else {
			newpc = pc.Increment(instructionSize)
		}

	case 0xD2:
		// RET
		if execute {
			newpc, err = proc.TopPop()
			if err != nil {
				return vStack, flags, syscall, err
			}
		} else {
			newpc = pc.Increment(instructionSize)
		}

	default:
		// invalid opcode
		s := fmt.Sprintf("Invalid opcode %02x at %s\n", opcode, pc.ToString())
		return vStack, flags, 0, errors.New(s)
	}

	// advance to next instruction
	err = proc.SetPC(newpc)
	if err != nil {
		s := fmt.Sprintf("Invalid address %s for PC in main: %s", newpc.ToString(), err.Error())
		return vStack, flags, 0, errors.New(s)
	}

	return vStack, flags, syscall, err
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
}

// Init - initialize
func (mod *Module) Init() {
}

// Write a module to a file
func (mod Module) Write(filename string) error {
	f, err := os.Create(filename)
	if err != nil {
		return err
	}

	defer f.Close()

	vputils.WriteString(f, "module")

	vputils.WriteTextTable("properties", mod.Properties, f)
	vputils.WriteTextTable("exports", mod.Exports, f)
	vputils.WriteTextTable("code_properties", mod.CodePage.Properties, f)
	vputils.WriteBinaryBlock("code", mod.CodePage.Contents, f, mod.CodeAddressWidth)
	vputils.WriteTextTable("data_properties", mod.DataPage.Properties, f)
	vputils.WriteBinaryBlock("data", mod.DataPage.Contents, f, mod.DataAddressWidth)

	f.Sync()

	return nil
}

// Read a file into a module
func Read(moduleFile string) (Module, error) {
	f, err := os.Open(moduleFile)
	if err != nil {
		return Module{}, err
	}

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
	if err != nil {
		return Module{}, err
	}

	codeAddressWidth := 1
	dataAddressWidth := 1

	header = vputils.ReadString(f)
	if header != "exports" {
		return Module{}, errors.New("Did not find exports header")
	}

	exports, err := vputils.ReadTextTable(f)
	if err != nil {
		return Module{}, err
	}

	header = vputils.ReadString(f)
	if header != "code_properties" {
		return Module{}, errors.New("Did not find code_properties header")
	}

	codeProperties, err := vputils.ReadTextTable(f)
	if err != nil {
		return Module{}, err
	}

	header = vputils.ReadString(f)
	if header != "code" {
		return Module{}, errors.New("Did not find code header")
	}

	code, err := vputils.ReadBinaryBlock(f, codeAddressWidth)
	if err != nil {
		return Module{}, err
	}

	codePage := Page{codeProperties, code}

	header = vputils.ReadString(f)
	if header != "data_properties" {
		return Module{}, errors.New("Did not find data_properties header")
	}

	dataProperties, err := vputils.ReadTextTable(f)
	if err != nil {
		return Module{}, err
	}

	header = vputils.ReadString(f)
	if header != "data" {
		return Module{}, errors.New("Did not find data header")
	}

	data, err := vputils.ReadBinaryBlock(f, dataAddressWidth)
	if err != nil {
		return Module{}, err
	}

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

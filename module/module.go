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

// MnemonicTargetWidthAddressMode
type MnemonicTargetWidthAddressMode struct {
	Name        string
	Width       string
	AddressMode string
}

// ToString ----------------------
func (def MnemonicTargetWidthAddressMode) ToString() string {
	s := def.Name

	if len(def.Width) > 0 {
		s += " "
		s += def.Width
	}

	return s
}

// OpcodeSize --------------------
func (def MnemonicTargetWidthAddressMode) OpcodeSize() int {
	return 1
}

// TargetSize --------------------
func (def MnemonicTargetWidthAddressMode) TargetSize() int {
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

// ByteToMnemonic ----------------
type ByteToMnemonic map[byte]MnemonicTargetWidthAddressMode

// DefineOpcodes - define the table of opcodes
func DefineOpcodes() ByteToMnemonic {
	bytesToMnemonics := make(ByteToMnemonic)

	bytesToMnemonics[0x00] = MnemonicTargetWidthAddressMode{"NOP", "", ""}
	bytesToMnemonics[0x04] = MnemonicTargetWidthAddressMode{"EXIT", "", ""}
	bytesToMnemonics[0x05] = MnemonicTargetWidthAddressMode{"KCALL", "", ""}
	bytesToMnemonics[0x08] = MnemonicTargetWidthAddressMode{"OUT", "", "S"}

	bytesToMnemonics[0x60] = MnemonicTargetWidthAddressMode{"PUSH", "BYTE", "V"}
	bytesToMnemonics[0x61] = MnemonicTargetWidthAddressMode{"PUSH", "BYTE", "D"}
	bytesToMnemonics[0x62] = MnemonicTargetWidthAddressMode{"PUSH", "BYTE", "I"}

	bytesToMnemonics[0x64] = MnemonicTargetWidthAddressMode{"PUSH", "I16", "V"}
	bytesToMnemonics[0x65] = MnemonicTargetWidthAddressMode{"PUSH", "I16", "D"}
	bytesToMnemonics[0x66] = MnemonicTargetWidthAddressMode{"PUSH", "I16", "I"}

	bytesToMnemonics[0x79] = MnemonicTargetWidthAddressMode{"PUSH", "STRING", "D"}

	bytesToMnemonics[0x81] = MnemonicTargetWidthAddressMode{"POP", "BYTE", "D"}
	bytesToMnemonics[0x82] = MnemonicTargetWidthAddressMode{"POP", "BYTE", "I"}
	bytesToMnemonics[0x83] = MnemonicTargetWidthAddressMode{"POP", "BYTE", "S"}

	bytesToMnemonics[0x11] = MnemonicTargetWidthAddressMode{"FLAGS", "BYTE", "D"}
	bytesToMnemonics[0x12] = MnemonicTargetWidthAddressMode{"FLAGS", "BYTE", "I"}
	bytesToMnemonics[0x13] = MnemonicTargetWidthAddressMode{"FLAGS", "BYTE", "S"}

	bytesToMnemonics[0x21] = MnemonicTargetWidthAddressMode{"INC", "BYTE", "D"}
	bytesToMnemonics[0x22] = MnemonicTargetWidthAddressMode{"INC", "BYTE", "I"}
	bytesToMnemonics[0x31] = MnemonicTargetWidthAddressMode{"DEC", "BYTE", "D"}
	bytesToMnemonics[0x32] = MnemonicTargetWidthAddressMode{"DEC", "BYTE", "I"}

	bytesToMnemonics[0xD0] = MnemonicTargetWidthAddressMode{"JUMP", "", ""}
	bytesToMnemonics[0xD1] = MnemonicTargetWidthAddressMode{"CALL", "", ""}
	bytesToMnemonics[0xD2] = MnemonicTargetWidthAddressMode{"RET", "", ""}

	bytesToMnemonics[0xA0] = MnemonicTargetWidthAddressMode{"ADD", "BYTE", ""}
	bytesToMnemonics[0xA1] = MnemonicTargetWidthAddressMode{"SUB", "BYTE", ""}
	bytesToMnemonics[0xA2] = MnemonicTargetWidthAddressMode{"MUL", "BYTE", ""}
	bytesToMnemonics[0xA3] = MnemonicTargetWidthAddressMode{"DIV", "BYTE", ""}

	bytesToMnemonics[0xC0] = MnemonicTargetWidthAddressMode{"AND", "BYTE", ""}
	bytesToMnemonics[0xC1] = MnemonicTargetWidthAddressMode{"OR", "BYTE", ""}
	bytesToMnemonics[0xC3] = MnemonicTargetWidthAddressMode{"CMP", "BYTE", ""}

	return bytesToMnemonics
}

type TargetWidthToOpcodes map[string][]byte

type OpcodeBytes struct {
	Opcode         byte
	AddressOpcodes TargetWidthToOpcodes
}

func MakeMnemonicTargetWidthAddressModes() map[string]OpcodeBytes {
	opcodeDefs := map[string]OpcodeBytes{}

	emptyOpcodes := make(TargetWidthToOpcodes)

	opcodeDefs["NOP"] = OpcodeBytes{0x00, emptyOpcodes}
	opcodeDefs["EXIT"] = OpcodeBytes{0x04, emptyOpcodes}
	opcodeDefs["KCALL"] = OpcodeBytes{0x05, emptyOpcodes}
	opcodeDefs["OUT"] = OpcodeBytes{0x08, emptyOpcodes}

	opcodeDefs["JUMP"] = OpcodeBytes{0xD0, emptyOpcodes}

	opcodeDefs["CALL"] = OpcodeBytes{0xD1, emptyOpcodes}

	opcodeDefs["RET"] = OpcodeBytes{0xD2, emptyOpcodes}

	pushOpcodes := make(TargetWidthToOpcodes)
	pushOpcodes["BYTE"] = []byte{0x60, 0x61, 0x62, 0x0F}
	pushOpcodes["I16"] = []byte{0x64, 0x65, 0x66, 0x0F}
	pushOpcodes["STRING"] = []byte{0x0F, 0x79, 0x7A, 0x0F}
	opcodeDefs["PUSH"] = OpcodeBytes{0x0F, pushOpcodes}

	popOpcodes := make(TargetWidthToOpcodes)
	popOpcodes["BYTE"] = []byte{0x0F, 0x81, 0x82, 0x83}
	opcodeDefs["POP"] = OpcodeBytes{0x0F, popOpcodes}

	flagsOpcodes := make(TargetWidthToOpcodes)
	flagsOpcodes["BYTE"] = []byte{0x10, 0x11, 0x12, 0x13}
	opcodeDefs["FLAGS"] = OpcodeBytes{0x0F, flagsOpcodes}

	incOpcodes := make(TargetWidthToOpcodes)
	incOpcodes["BYTE"] = []byte{0x0F, 0x21, 0x22, 0x23}
	opcodeDefs["INC"] = OpcodeBytes{0x0F, incOpcodes}

	decOpcodes := make(TargetWidthToOpcodes)
	decOpcodes["BYTE"] = []byte{0x0F, 0x31, 0x32, 0x33}
	opcodeDefs["DEC"] = OpcodeBytes{0x0F, decOpcodes}

	addOpcodes := make(TargetWidthToOpcodes)
	addOpcodes["BYTE"] = []byte{0x0F, 0x0F, 0x0F, 0xA0}
	opcodeDefs["ADD"] = OpcodeBytes{0x0F, addOpcodes}

	subOpcodes := make(TargetWidthToOpcodes)
	subOpcodes["BYTE"] = []byte{0x0F, 0x0F, 0x0F, 0xA1}
	opcodeDefs["SUB"] = OpcodeBytes{0x0F, subOpcodes}

	mulOpcodes := make(TargetWidthToOpcodes)
	mulOpcodes["BYTE"] = []byte{0x0F, 0x0F, 0x0F, 0xA2}
	opcodeDefs["MUL"] = OpcodeBytes{0x0F, mulOpcodes}

	divOpcodes := make(TargetWidthToOpcodes)
	divOpcodes["BYTE"] = []byte{0x0F, 0x0F, 0x0F, 0xA3}
	opcodeDefs["DIV"] = OpcodeBytes{0x0F, divOpcodes}

	andOpcodes := make(TargetWidthToOpcodes)
	andOpcodes["BYTE"] = []byte{0x0F, 0x0F, 0x0F, 0xC0}
	opcodeDefs["AND"] = OpcodeBytes{0x0F, andOpcodes}

	orOpcodes := make(TargetWidthToOpcodes)
	orOpcodes["BYTE"] = []byte{0x0F, 0x0F, 0x0F, 0xC1}
	opcodeDefs["OR"] = OpcodeBytes{0x0F, orOpcodes}

	cmpOpcodes := make(TargetWidthToOpcodes)
	cmpOpcodes["BYTE"] = []byte{0x0F, 0x0F, 0x0F, 0xC3}
	opcodeDefs["CMP"] = OpcodeBytes{0x0F, cmpOpcodes}

	return opcodeDefs
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
	Flags    FlagsGroup
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

// IncrementPC - increment the PC
func (proc *Processor) IncrementPC(count int) {
	proc.pc = proc.pc.Increment(count)
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

func (proc Processor) DecodeInstruction(opcode byte, def MnemonicTargetWidthAddressMode, code Page, data Page) (InstructionDefinition, error) {
	fullOpcode := []byte{opcode}

	// working bytes for opcode
	workBytes := []byte{}

	// addresses for opcode
	dataAddress, _ := vputils.MakeAddress(0, 0, 0)
	dataAddress1, _ := vputils.MakeAddress(0, 0, 0)
	jumpAddress, _ := vputils.MakeAddress(0, 0, 0)
	valueStr := ""

	instructionSize := def.OpcodeSize()
	targetSize := def.TargetSize()

	err := errors.New("")

	// decode immediate value
	if def.AddressMode == "V" {

		switch def.Width {

		case "BYTE":
			workBytes, err = code.ImmediateByte(proc.PC())
			if err != nil {
				return InstructionDefinition{}, err
			}

			valueStr = fmt.Sprintf("%02X", workBytes[0])

		case "I16":
			workBytes, err = code.ImmediateInt(proc.PC())
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
		dataAddress, err = code.DirectAddress(proc.PC(), data)
		if err != nil {
			return InstructionDefinition{}, err
		}

		bytes := dataAddress.ToBytes()
		fullOpcode = append(fullOpcode, bytes...)

		buffer, err := code.DirectByte(proc.PC(), data)
		if err != nil {
			return InstructionDefinition{}, err
		}

		workBytes = append(workBytes, buffer)
		valueStr = fmt.Sprintf("%02X", buffer)

		instructionSize += dataAddress.Size
	}

	if def.AddressMode == "I" {
		dataAddress1, err = code.DirectAddress(proc.PC(), data)
		if err != nil {
			return InstructionDefinition{}, err
		}

		bytes1 := dataAddress1.ToBytes()
		fullOpcode = append(fullOpcode, bytes1...)
		bytes := dataAddress.ToBytes()
		workBytes = append(workBytes, bytes...)

		dataAddress, err = code.IndirectAddress(proc.PC(), data)
		if err != nil {
			return InstructionDefinition{}, err
		}

		buffer, err := code.IndirectByte(proc.PC(), data)
		if err != nil {
			return InstructionDefinition{}, err
		}

		workBytes = append(workBytes, buffer)
		valueStr = fmt.Sprintf("%02X", buffer)

		instructionSize += dataAddress1.Size
	}

	// decode jump/call target
	if opcode == 0xD0 || opcode == 0xD1 {
		jumpAddress, err = code.JumpAddress(proc.PC())
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
func (proc *Processor) ExecuteOpcode(data *Page, opcode byte, vStack vputils.ByteStack, instruction InstructionDefinition, execute bool) (vputils.ByteStack, byte, error) {
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
			proc.Flags.Zero = bytes[0] == 0
		}

		newpc = pc.Increment(instructionSize)

	case 0x12:
		// FLAGS.B indirect address
		if execute {
			proc.Flags.Zero = bytes[0] == 0
		}

		newpc = pc.Increment(instructionSize)

	case 0x13:
		// FLAGS.B (implied stack)
		if execute {
			buffer, err := vStack.TopByte()
			if err != nil {
				return vStack, syscall, err
			}

			proc.Flags.Zero = buffer == 0
		}

		newpc = pc.Increment(instructionSize)

	case 0x21:
		// INC.B direct address
		if execute {
			bytes[0]++

			err = data.Contents.PutByte(dataAddress, bytes[0])
			if err != nil {
				return vStack, syscall, err
			}
		}

		newpc = pc.Increment(instructionSize)

	case 0x22:
		// INC.B indirect address
		if execute {
			bytes[0]++

			err = data.Contents.PutByte(dataAddress, bytes[0])
			if err != nil {
				return vStack, syscall, err
			}
		}

		newpc = pc.Increment(instructionSize)

	case 0x31:
		// DEC.B direct address
		if execute {
			bytes[0]--

			err = data.Contents.PutByte(dataAddress, bytes[0])
			if err != nil {
				return vStack, syscall, err
			}
		}

		newpc = pc.Increment(instructionSize)

	case 0x32:
		// DEC.B indirect address
		if execute {
			bytes[0]--

			err = data.Contents.PutByte(dataAddress, bytes[0])
			if err != nil {
				return vStack, syscall, err
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
					return vStack, syscall, err
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
				return vStack, syscall, err
			}

			err = data.Contents.PutByte(dataAddress, bytes[0])
			if err != nil {
				return vStack, syscall, err
			}
		}

		newpc = pc.Increment(instructionSize)

	case 0x83:
		// POP.B value (to nowhere)
		if execute {
			bytes, vStack, err = vStack.PopByte(1)
			if err != nil {
				return vStack, syscall, err
			}
		}

		newpc = pc.Increment(instructionSize)

	case 0xA0:
		// ADD.B
		if execute {
			bytes1, vStack, err = vStack.PopByte(1)
			if err != nil {
				return vStack, syscall, err
			}

			bytes2, vStack, err = vStack.PopByte(1)
			if err != nil {
				return vStack, syscall, err
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
				return vStack, syscall, err
			}

			bytes2, vStack, err = vStack.PopByte(1)
			if err != nil {
				return vStack, syscall, err
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
				return vStack, syscall, err
			}

			bytes2, vStack, err = vStack.PopByte(1)
			if err != nil {
				return vStack, syscall, err
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
				return vStack, syscall, err
			}

			bytes2, vStack, err = vStack.PopByte(1)
			if err != nil {
				return vStack, syscall, err
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
				return vStack, syscall, err
			}

			bytes2, vStack, err = vStack.PopByte(1)
			if err != nil {
				return vStack, syscall, err
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
				return vStack, syscall, err
			}

			bytes2, vStack, err = vStack.PopByte(1)
			if err != nil {
				return vStack, syscall, err
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
				return vStack, syscall, err
			}

			bytes2, vStack, err = vStack.PopByte(1)
			if err != nil {
				return vStack, syscall, err
			}

			value := bytes1[0] - bytes2[0]

			proc.Flags.Zero = value == 0
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
				return vStack, syscall, err
			}
		} else {
			newpc = pc.Increment(instructionSize)
		}

	default:
		// invalid opcode
		s := fmt.Sprintf("Invalid opcode %02x at %s\n", opcode, pc.ToString())
		return vStack, 0, errors.New(s)
	}

	// advance to next instruction
	err = proc.SetPC(newpc)
	if err != nil {
		s := fmt.Sprintf("Invalid address %s for PC in main: %s", newpc.ToString(), err.Error())
		return vStack, 0, errors.New(s)
	}

	return vStack, syscall, err
}

func traceOpcode(pc vputils.Address, opcode byte, opcodeDef MnemonicTargetWidthAddressMode, flags FlagsGroup, conditionals Conditionals, instruction InstructionDefinition) string {
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

// ExecuteInstruction - execute an instruction
func (proc *Processor) ExecuteInstruction(vStack vputils.ByteStack, codePage Page, dataPage *Page, trace bool) (vputils.ByteStack, byte, error) {
	opcodeDefinitions := DefineOpcodes()

	pc1 := proc.PC()

	conditionals, err := codePage.GetConditionals(pc1)
	if err != nil {
		message := err.Error() + " at PC " + pc1.ToString()
		return vStack, 0, errors.New(message)
	}

	proc.IncrementPC(len(conditionals))

	execute, err := conditionals.Evaluate(proc.Flags)
	if err != nil {
		return vStack, 0, err
	}

	pc2 := proc.PC()

	opcode, err := codePage.GetOpcode(pc2)
	if err != nil {
		message := err.Error() + " at PC " + pc2.ToString()
		return vStack, 0, errors.New(message)
	}

	def := opcodeDefinitions[opcode]

	// get instruction definition (opcode and arguments)
	instruction, err := proc.DecodeInstruction(opcode, def, codePage, *dataPage)
	vputils.CheckAndExit(err)

	if trace {
		line := traceOpcode(pc1, opcode, def, proc.Flags, conditionals, instruction)
		fmt.Println(line)
	}

	// execute instruction
	syscall := byte(0)

	vStack, syscall, err = proc.ExecuteOpcode(dataPage, opcode, vStack, instruction, execute)

	return vStack, syscall, err
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

	codeAddressWidth := 1

	header = vputils.ReadString(f)
	if header != "code" {
		return Module{}, errors.New("Did not find code header")
	}

	code, err := vputils.ReadBinaryBlock(f, codeAddressWidth)
	if err != nil {
		return Module{}, err
	}

	codePage := Page{codeProperties, code, codeAddressWidth}

	header = vputils.ReadString(f)
	if header != "data_properties" {
		return Module{}, errors.New("Did not find data_properties header")
	}

	dataProperties, err := vputils.ReadTextTable(f)
	if err != nil {
		return Module{}, err
	}

	dataAddressWidth := 1

	header = vputils.ReadString(f)
	if header != "data" {
		return Module{}, errors.New("Did not find data header")
	}

	data, err := vputils.ReadBinaryBlock(f, dataAddressWidth)
	if err != nil {
		return Module{}, err
	}

	dataPage := Page{dataProperties, data, dataAddressWidth}

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

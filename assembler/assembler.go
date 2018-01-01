/*
package main of assembler
*/
package main

import (
	"errors"
	"fmt"
	"github.com/jfitz/virtual-processor/vputils"
	"os"
	"strconv"
	"strings"
)

func first(tokens []string) (string, []string) {
	firstToken := ""

	// get the leading token if it is not whitespace
	if len(tokens) > 0 && len(tokens[0]) > 0 && !vputils.IsSpace(tokens[0][0]) {
		firstToken = tokens[0]
		tokens = tokens[1:]
	}

	// skip the whitespace
	if len(tokens) > 0 && len(tokens[0]) > 0 && vputils.IsSpace(tokens[0][0]) {
		tokens = tokens[1:]
	}

	return firstToken, tokens
}

func isDirective(s string) bool {
	if s == "BYTE" {
		return true
	}

	if s == "STRING" {
		return true
	}

	return false
}

func evaluateByte(expression string) []byte {
	value, err := strconv.Atoi(expression)
	vputils.CheckAndPanic(err)

	byteValue := byte(value)

	return []byte{byteValue}
}

func evaluateI16(expression string) []byte {
	value, err := strconv.Atoi(expression)
	vputils.CheckAndPanic(err)

	byteValue1 := byte(value & 0xff)
	byteValue2 := byte((value >> 8) & 0xff)

	return []byte{byteValue1, byteValue2}
}

type LabelTable map[string]vputils.Address

func buildInstruction(opcodemap opcodeList, targetType string, target string, dataLabels LabelTable) ([]byte, error) {
	opcodes, ok := opcodemap[targetType]

	if !ok {
		return nil, errors.New("Set '" + targetType + "' not found")
	}

	if len(target) == 0 {
		// stack
		opcode := opcodes[3]
		instruction := []byte{opcode}
		return instruction, nil
	}

	if vputils.IsDigit(target[0]) {
		// immediate value
		opcode := []byte{opcodes[0]}
		bytes := []byte{}
		switch targetType {
		case "B":
			bytes = evaluateByte(target)
		case "I16":
			bytes = evaluateI16(target)
		}
		instruction := append(opcode, bytes...)
		return instruction, nil
	}

	if vputils.IsAlpha(target[0]) {
		// immediate value
		opcode := []byte{opcodes[0]}
		address, ok := dataLabels[target]
		if !ok {
			err := errors.New("Undefined label '" + target + "'")
			vputils.CheckAndExit(err)
		}
		instruction := append(opcode, address.Bytes...)
		return instruction, nil
	}

	if vputils.IsDirectAddress(target) {
		// direct address
		opcode := []byte{opcodes[1]}
		address, ok := dataLabels[target[1:]]
		if !ok {
			err := errors.New("Undefined label '" + target + "'")
			vputils.CheckAndExit(err)
		}
		instruction := append(opcode, address.Bytes...)
		return instruction, nil
	}

	if vputils.IsIndirectAddress(target) {
		// indirect address
		opcode := []byte{opcodes[2]}
		address, ok := dataLabels[target[2:]]
		if !ok {
			err := errors.New("Undefined label '" + target + "'")
			vputils.CheckAndExit(err)
		}
		instruction := append(opcode, address.Bytes...)
		return instruction, nil
	}

	return nil, errors.New("Invalid opcode")
}

type opcodeList map[string][]byte

type opcodeDefinition struct {
	Opcode         byte
	AddressOpcodes opcodeList
	JumpOpcodes    opcodeList
}

func decodeOpcode(text string, instructionAddress vputils.Address, targetType string, target string, opcodeDefs map[string]opcodeDefinition, resolveAddress bool, codeLabels LabelTable, dataLabels LabelTable) ([]byte, error) {
	opcodeDef, ok := opcodeDefs[text]

	if !ok {
		return []byte{}, errors.New("Invalid opcode: '" + text + "' ")
	}

	// assume we have a simple opcode (with no target)
	instruction := []byte{opcodeDef.Opcode}
	addressOpcodes := opcodeDef.AddressOpcodes
	jumpOpcodes := opcodeDef.JumpOpcodes

	var err error
	if len(addressOpcodes) > 0 {
		// select instruction depends on target
		instruction, err = buildInstruction(addressOpcodes, targetType, target, dataLabels)
		vputils.CheckAndExit(err)
	}

	// for jump instructions, append the target address
	if len(jumpOpcodes) > 0 {
		address, ok := codeLabels[target]
		if !ok {
			if resolveAddress {
				err = errors.New("Undefined label '" + target + "'")
				vputils.CheckAndExit(err)
			} else {
				address = vputils.MakeAddress(0, 1)
			}
		}

		opcodes, ok := jumpOpcodes[targetType]
		if !ok {
			return nil, errors.New("Decode: Set '" + targetType + "' not found for " + text)
		}

		instruction = []byte{opcodes[0]}

		if targetType == "A" {
			instruction = append(instruction, address.Bytes...)
		}
		if targetType == "R" {
			offset := byte(address.ToInt() - instructionAddress.ToInt())
			instruction = append(instruction, offset)
		}
	}

	return instruction, nil
}

func getInstruction(text string, instructionAddress vputils.Address, targetType string, target string, opcodeDefs map[string]opcodeDefinition, resolveAddress bool, dataLabels LabelTable, codeLabels LabelTable) []byte {
	instruction, err := decodeOpcode(text, instructionAddress, targetType, target, opcodeDefs, resolveAddress, codeLabels, dataLabels)
	vputils.CheckAndExit(err)

	if len(instruction) == 0 {
		err := errors.New("Empty opcode")
		vputils.CheckAndExit(err)
	}

	return instruction
}

func checkDataLabel(label string, labels LabelTable) {
	if label == "" {
		vputils.CheckAndExit(errors.New("Data declaration requires label"))
	}

	if _, ok := labels[label]; ok {
		vputils.CheckAndExit(errors.New("Duplicate label " + label))
	}
}

func checkCodeLabel(label string, labels LabelTable) {
	if _, ok := labels[label]; ok {
		vputils.CheckAndExit(errors.New("Duplicate label " + label))
	}
}

func printLabels(labels LabelTable) {
	for k, v := range labels {
		fmt.Printf("%s\t%d\n", k, v)
	}
}

func dequoteString(s string) []byte {
	last := len(s) - 1
	s = s[1:last]
	bytes := []byte{}
	for _, c := range s {
		bytes = append(bytes, byte(c))
	}
	bytes = append(bytes, byte(0))

	return bytes
}

func generateData(source []string, opcodeDefs map[string]opcodeDefinition) (vputils.Vector, LabelTable, LabelTable) {
	tabs := "\t\t\t"
	fmt.Println(tabs + "DATA")

	code := vputils.Vector{}
	codeLabels := make(LabelTable)
	data := vputils.Vector{}
	dataLabels := make(LabelTable)

	for _, line := range source {
		// remove comment from line
		// remove trailing whitespace
		line = strings.TrimRight(line, " \t")
		// only lines with content
		if len(line) > 0 {
			tokens := vputils.Tokenize(line)
			label, tokens := first(tokens)
			word, tokens := first(tokens)

			// write the directive or instruction
			if isDirective(word) {
				checkDataLabel(label, dataLabels)

				// add the label to our table
				address := len(data)
				if address > 255 {
					vputils.CheckAndExit(errors.New("Exceeded data label table size"))
				}
				dataLabels[label] = vputils.MakeAddress(address, 1)

				// write the label on a line by itself
				if len(label) > 0 {
					fmt.Printf("%s:\n", label)
				}

				values := []byte{}
				switch word {
				case "BYTE":
					target, _ := first(tokens)
					// evaluate numeric or text (data label) but nothing else
					value := evaluateByte(target)
					values = append(values, value...)
				case "STRING":
					target, _ := first(tokens)
					// target must be a string
					chars := dequoteString(target)
					values = append(values, chars...)
				default:
					vputils.CheckAndExit(errors.New("Invalid directive " + word))
				}

				// print offset, directive, and contents
				location := len(data)

				if len(values) == 0 {
					fmt.Printf("%02X%s%s\n", location, tabs, word)
				} else {
					fmt.Printf("%02X%s%s\t\t% X\n", location, tabs, word, values)
					data = append(data, values...)
				}
			} else {
				// process instruction
				opcode := word
				targetType := ""
				parts := strings.Split(word, ".")
				if len(parts) > 1 {
					opcode = parts[0]
					targetType = parts[1]
				}

				address := len(code)
				if address > 255 {
					vputils.CheckAndExit(errors.New("Exceeded code label table size"))
				}
				instructionAddress := vputils.MakeAddress(address, 1)

				if len(label) > 0 {
					// add the label to our table
					checkCodeLabel(label, codeLabels)
					codeLabels[label] = instructionAddress
				}

				target, tokens := first(tokens)

				// check there are no more tokens
				if len(tokens) > 0 {
					fmt.Println(line)
					vputils.CheckAndExit(errors.New("Extra tokens on line"))
				}

				// decode the instruction
				instruction := getInstruction(opcode, instructionAddress, targetType, target, opcodeDefs, false, dataLabels, codeLabels)

				code = append(code, instruction...)
			}
		}
	}

	fmt.Println(tabs + "ENDSEGMENT")
	fmt.Println()

	return data, dataLabels, codeLabels
}

func generateCode(source []string, opcodeDefs map[string]opcodeDefinition, dataLabels LabelTable, codeLabels LabelTable) vputils.Vector {
	tabs := "\t\t\t"
	fmt.Println(tabs + "CODE")

	code := vputils.Vector{}

	for _, line := range source {
		// remove comment from line
		// remove trailing whitespace
		line = strings.TrimRight(line, " \t")
		// only lines with content
		if len(line) > 0 {
			tokens := vputils.Tokenize(line)
			label, tokens := first(tokens)
			word, tokens := first(tokens)

			// write the directive or instruction
			if !isDirective(word) {
				opcode := word
				targetType := ""
				parts := strings.Split(word, ".")
				if len(parts) > 1 {
					opcode = parts[0]
					targetType = parts[1]
				}

				address := len(code)
				if address > 255 {
					vputils.CheckAndExit(errors.New("Exceeded code label table size"))
				}
				instructionAddress := vputils.MakeAddress(address, 1)

				if len(label) > 0 {
					// write the label on a line by itself
					fmt.Printf("%s:\n", label)
				}

				target, tokens := first(tokens)

				// check that there are no more tokens
				if len(tokens) > 0 {
					vputils.CheckAndExit(errors.New("Extra tokens on line"))
				}

				// decode the instruction
				instruction := getInstruction(opcode, instructionAddress, targetType, target, opcodeDefs, true, dataLabels, codeLabels)

				location := len(code)
				instruction_s := fmt.Sprintf("% X", instruction)
				wordTabCount := 2 - len(instruction_s)/8
				wordTabs := ""
				for i := 0; i < wordTabCount; i++ {
					wordTabs += "\t"
				}

				fmt.Printf("%02X\t%s%s%s\t%s\n", location, instruction_s, wordTabs, word, target)

				code = append(code, instruction...)
			}
		}
	}
	fmt.Println(tabs + "ENDSEGMENT")
	fmt.Println()

	return code
}

func (module vputils.Module) write(filename string) {
	f, err := os.Create(filename)
	vputils.CheckAndPanic(err)

	defer f.Close()

	vputils.WriteString(f, "module")

	vputils.WriteTextTable("properties", module.Properties, f)
	vputils.WriteTextTable("exports", module.Exports, f)
	vputils.WriteBinaryBlock("code", module.Code, f, module.CodeAddressWidth)
	vputils.WriteBinaryBlock("data", module.Data, f, module.DataAddressWidth)

	f.Sync()
}

func makeProperties(instructionSetVersion string, codeAddressWidth int, dataAddressWidth int) []vputils.NameValue {
	caws := strconv.Itoa(codeAddressWidth)
	daws := strconv.Itoa(dataAddressWidth)

	properties := []vputils.NameValue{}

	properties = append(properties, vputils.NameValue{"INSTRUCTION SET VERSION", instructionSetVersion})
	properties = append(properties, vputils.NameValue{"STACK WIDTH", "1"})
	properties = append(properties, vputils.NameValue{"DATA WIDTH", "1"})
	properties = append(properties, vputils.NameValue{"ADDRESS WIDTH", "1"})
	properties = append(properties, vputils.NameValue{"CODE ADDRESS WIDTH", caws})
	properties = append(properties, vputils.NameValue{"DATA ADDRESS WIDTH", daws})
	properties = append(properties, vputils.NameValue{"CALL STACK SIZE", "1"})

	return properties
}

func makeOpcodeDefinitions() map[string]opcodeDefinition {
	opcodeDefs := map[string]opcodeDefinition{}

	empty_opcodes := make(opcodeList)

	opcodeDefs["EXIT"] = opcodeDefinition{0x00, empty_opcodes, empty_opcodes}
	opcodeDefs["OUT"] = opcodeDefinition{0x08, empty_opcodes, empty_opcodes}

	jump_opcodes := make(opcodeList)
	jump_opcodes["A"] = []byte{0xD0}
	jump_opcodes["R"] = []byte{0xE0}
	jnz_opcodes := make(opcodeList)
	jnz_opcodes["A"] = []byte{0xD1}
	jnz_opcodes["R"] = []byte{0xE1}
	jz_opcodes := make(opcodeList)
	jz_opcodes["A"] = []byte{0xD2}
	jz_opcodes["R"] = []byte{0xE2}
	opcodeDefs["JUMP"] = opcodeDefinition{0x0F, empty_opcodes, jump_opcodes}
	opcodeDefs["JNZ"] = opcodeDefinition{0x0F, empty_opcodes, jnz_opcodes}
	opcodeDefs["JZ"] = opcodeDefinition{0x0F, empty_opcodes, jz_opcodes}

	call_opcodes := make(opcodeList)
	call_opcodes["A"] = []byte{0xD4}
	call_opcodes["R"] = []byte{0xE4}
	cz_opcodes := make(opcodeList)
	cz_opcodes["A"] = []byte{0xD6}
	cz_opcodes["R"] = []byte{0xE6}
	opcodeDefs["CALL"] = opcodeDefinition{0x0F, empty_opcodes, call_opcodes}
	opcodeDefs["CZ"] = opcodeDefinition{0x0F, empty_opcodes, cz_opcodes}
	opcodeDefs["RET"] = opcodeDefinition{0xD8, empty_opcodes, empty_opcodes}
	opcodeDefs["RZ"] = opcodeDefinition{0xDA, empty_opcodes, empty_opcodes}

	push_opcodes := make(opcodeList)
	push_opcodes["B"] = []byte{0x60, 0x61, 0x62, 0x0F}
	push_opcodes["I16"] = []byte{0x64, 0x65, 0x66, 0x0F}
	push_opcodes["STR"] = []byte{0x0F, 0x79, 0x7A, 0x0F}
	opcodeDefs["PUSH"] = opcodeDefinition{0x0F, push_opcodes, empty_opcodes}

	pop_opcodes := make(opcodeList)
	pop_opcodes["B"] = []byte{0x0F, 0x81, 0x82, 0x0F}
	opcodeDefs["POP"] = opcodeDefinition{0x0F, pop_opcodes, empty_opcodes}

	flags_opcodes := make(opcodeList)
	flags_opcodes["B"] = []byte{0x10, 0x11, 0x12, 0x13}
	opcodeDefs["FLAGS"] = opcodeDefinition{0x0F, flags_opcodes, empty_opcodes}

	inc_opcodes := make(opcodeList)
	inc_opcodes["B"] = []byte{0x0F, 0x21, 0x22, 0x23}
	opcodeDefs["INC"] = opcodeDefinition{0x0F, inc_opcodes, empty_opcodes}

	dec_opcodes := make(opcodeList)
	dec_opcodes["B"] = []byte{0x0F, 0x31, 0x32, 0x33}
	opcodeDefs["DEC"] = opcodeDefinition{0x0F, dec_opcodes, empty_opcodes}

	add_opcodes := make(opcodeList)
	add_opcodes["B"] = []byte{0x0F, 0x0F, 0x0F, 0xA0}
	opcodeDefs["ADD"] = opcodeDefinition{0x0F, add_opcodes, empty_opcodes}

	sub_opcodes := make(opcodeList)
	sub_opcodes["B"] = []byte{0x0F, 0x0F, 0x0F, 0xA1}
	opcodeDefs["SUB"] = opcodeDefinition{0x0F, sub_opcodes, empty_opcodes}

	mul_opcodes := make(opcodeList)
	mul_opcodes["B"] = []byte{0x0F, 0x0F, 0x0F, 0xA2}
	opcodeDefs["MUL"] = opcodeDefinition{0x0F, mul_opcodes, empty_opcodes}

	div_opcodes := make(opcodeList)
	div_opcodes["B"] = []byte{0x0F, 0x0F, 0x0F, 0xA3}
	opcodeDefs["DIV"] = opcodeDefinition{0x0F, div_opcodes, empty_opcodes}

	and_opcodes := make(opcodeList)
	and_opcodes["B"] = []byte{0x0F, 0x0F, 0x0F, 0xC0}
	opcodeDefs["AND"] = opcodeDefinition{0x0F, and_opcodes, empty_opcodes}

	or_opcodes := make(opcodeList)
	or_opcodes["B"] = []byte{0x0F, 0x0F, 0x0F, 0xC1}
	opcodeDefs["OR"] = opcodeDefinition{0x0F, or_opcodes, empty_opcodes}

	cmp_opcodes := make(opcodeList)
	cmp_opcodes["B"] = []byte{0x0F, 0x0F, 0x0F, 0xC3}
	opcodeDefs["CMP"] = opcodeDefinition{0x0F, cmp_opcodes, empty_opcodes}

	return opcodeDefs
}

func makeExports(codeLabels LabelTable) []vputils.NameValue {
	exports := []vputils.NameValue{}

	for label, address := range codeLabels {
		if vputils.IsUpper(label[0]) {
			i := address.ToInt()
			s := strconv.Itoa(i)
			nv := vputils.NameValue{label, s}
			exports = append(exports, nv)
		}
	}

	return exports
}

func main() {
	args := os.Args[1:]

	if len(args) == 0 {
		fmt.Println("No source file specified")
		os.Exit(1)
	}

	// read source
	sourceFile := args[0]
	source := vputils.ReadFile(sourceFile)

	// store output module file name
	moduleFile := ""
	if len(args) > 1 {
		moduleFile = args[1]
	}

	// create opcode definitions
	opcodeDefs := makeOpcodeDefinitions()
	instructionSetVersion := "1"

	codeAddressWidth := 1
	dataAddressWidth := 1

	properties := makeProperties(instructionSetVersion, codeAddressWidth, dataAddressWidth)

	data, dataLabels, codeLabels := generateData(source, opcodeDefs)

	exports := makeExports(codeLabels)

	code := generateCode(source, opcodeDefs, dataLabels, codeLabels)

	module := vputils.Module{
		Properties:       properties,
		Code:             code,
		Exports:          exports,
		Data:             data,
		CodeAddressWidth: codeAddressWidth,
		DataAddressWidth: dataAddressWidth,
	}

	// if output specified, write module file
	if len(moduleFile) > 0 {
		module.write(moduleFile)
	}
}

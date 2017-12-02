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

func evaluateByte(expression string) byte {
	value, err := strconv.Atoi(expression)
	vputils.CheckAndPanic(err)

	byteValue := byte(value)

	return byteValue
}

type LabelTable map[string]vputils.Address

func buildInstruction(opcodemap opcodeList, targetSize string, target string, dataLabels LabelTable) ([]byte, error) {
	opcodes, ok := opcodemap[targetSize]

	if !ok {
		return nil, errors.New("Set not found")
	}

	if len(target) == 0 {
		opcode := opcodes[3]
		return []byte{opcode}, nil
	}

	if vputils.IsDigit(target[0]) {
		opcode := opcodes[0]
		value := evaluateByte(target)
		return []byte{opcode, value}, nil
	}

	if vputils.IsAlpha(target[0]) {
		opcode := opcodes[0]
		instruction := []byte{opcode}
		address := dataLabels[target]
		instruction = append(instruction, address.Bytes...)
		return instruction, nil
	}

	if vputils.IsDirectAddress(target) {
		opcode := opcodes[1]
		instruction := []byte{opcode}
		address := dataLabels[target[1:]]
		instruction = append(instruction, address.Bytes...)
		return instruction, nil
	}

	if vputils.IsIndirectAddress(target) {
		opcode := opcodes[2]
		instruction := []byte{opcode}
		address := dataLabels[target[2:]]
		instruction = append(instruction, address.Bytes...)
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

func decodeOpcode(text string, instructionAddress vputils.Address, targetSize string, target string, opcodeDefs map[string]opcodeDefinition, codeLabels LabelTable, dataLabels LabelTable) ([]byte, error) {
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
		instruction, err = buildInstruction(addressOpcodes, targetSize, target, dataLabels)
		vputils.CheckAndExit(err)
	}

	// for jump instructions, append the target address
	if len(jumpOpcodes) > 0 {
		address, ok := codeLabels[target]
		if !ok {
			address = vputils.MakeAddress(0, 1)
		}

		opcodes, ok := jumpOpcodes[targetSize]
		if !ok {
			return nil, errors.New("Set not found")
		}

		if text == "JUMP" {
			instruction = []byte{opcodes[0]}
		}
		if text == "JZ" {
			instruction = []byte{opcodes[1]}
		}

		if targetSize == "A" {
			instruction = append(instruction, address.Bytes...)
		}
		if targetSize == "R" {
			offset := byte(address.ToInt() - instructionAddress.ToInt())
			instruction = append(instruction, offset)
		}
	}

	return instruction, nil
}

func getInstruction(text string, instructionAddress vputils.Address, targetSize string, target string, opcodeDefs map[string]opcodeDefinition, dataLabels LabelTable, codeLabels LabelTable) []byte {
	instruction, err := decodeOpcode(text, instructionAddress, targetSize, target, opcodeDefs, codeLabels, dataLabels)
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
	fmt.Println("\t\tDATA")

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
					values = append(values, value)
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
					fmt.Printf("%02X\t\t%s\n", location, word)
				} else {
					fmt.Printf("%02X\t\t%s\t\t% X\n", location, word, values)
					data = append(data, values...)
				}
			} else {
				// process instruction
				opcode := word
				targetSize := ""
				parts := strings.Split(word, ".")
				if len(parts) > 1 {
					opcode = parts[0]
					targetSize = parts[1]
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
					vputils.CheckAndExit(errors.New("Extra tokens on line"))
				}

				// decode the instruction
				instruction := getInstruction(opcode, instructionAddress, targetSize, target, opcodeDefs, dataLabels, codeLabels)

				code = append(code, instruction...)
			}
		}
	}
	fmt.Println("\t\tENDSEGMENT")
	fmt.Println()

	return data, dataLabels, codeLabels
}

func generateCode(source []string, opcodeDefs map[string]opcodeDefinition, dataLabels LabelTable, codeLabels LabelTable) vputils.Vector {
	fmt.Println("\t\tCODE")

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
				targetSize := ""
				parts := strings.Split(word, ".")
				if len(parts) > 1 {
					opcode = parts[0]
					targetSize = parts[1]
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
				instruction := getInstruction(opcode, instructionAddress, targetSize, target, opcodeDefs, dataLabels, codeLabels)

				location := len(code)

				fmt.Printf("%02X\t% X\t%s\t%s\n", location, instruction, word, target)

				code = append(code, instruction...)
			}
		}
	}
	fmt.Println("\t\tENDSEGMENT")
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
	jump_opcodes["A"] = []byte{0xD0, 0xD2}
	jump_opcodes["R"] = []byte{0xE0, 0xE2}
	opcodeDefs["JUMP"] = opcodeDefinition{0x0F, empty_opcodes, jump_opcodes}
	opcodeDefs["JZ"] = opcodeDefinition{0x0F, empty_opcodes, jump_opcodes}

	push_opcodes := make(opcodeList)
	push_opcodes["B"] = []byte{0x60, 0x61, 0x62, 0x0F}
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

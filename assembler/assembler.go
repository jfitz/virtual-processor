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

type Address struct {
	Bytes []byte
}

func (address Address) to_s() string {
	value := 0
	for _, b := range address.Bytes {
		value += int(b)
	}

	return strconv.Itoa(value)
}

type LabelTable map[string]Address

func buildInstruction(opcodemap opcodeAddresses, targetSize string, target string, dataLabels LabelTable) ([]byte, error) {
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

type opcodeAddresses map[string][]byte

type opcodeDefinition struct {
	Opcode         byte
	AddressOpcodes opcodeAddresses
	IsJump         bool
}

func decodeOpcode(text string, targetSize string, target string, opcodeDefs map[string]opcodeDefinition, codeLabels LabelTable, dataLabels LabelTable) ([]byte, error) {
	opcodeDef, ok := opcodeDefs[text]

	if !ok {
		return []byte{}, errors.New("Invalid opcode: '" + text + "' ")
	}

	// assume we have a simple opcode (with no target)
	instruction := []byte{opcodeDef.Opcode}
	addressOpcodes := opcodeDef.AddressOpcodes

	var err error
	if len(addressOpcodes) > 0 {
		// select instruction depends on target
		instruction, err = buildInstruction(addressOpcodes, targetSize, target, dataLabels)
		vputils.CheckAndPanic(err)
	}

	// for jump instructions, append the target address
	if opcodeDef.IsJump {
		address, ok := codeLabels[target]
		if !ok {
			address = Address{make([]byte, 1)}
		}
		instruction = append(instruction, address.Bytes...)
	}

	return instruction, nil
}

func getInstruction(text string, targetSize string, target string, opcodeDefs map[string]opcodeDefinition, dataLabels LabelTable, codeLabels LabelTable) []byte {
	instruction, err := decodeOpcode(text, targetSize, target, opcodeDefs, codeLabels, dataLabels)
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

func generateData(source []string, opcodeDefs map[string]opcodeDefinition) ([]byte, LabelTable, LabelTable) {
	fmt.Println("\t\tDATA")

	code := []byte{}
	codeLabels := make(LabelTable)
	data := []byte{}
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
				dataLabels[label] = Address{[]byte{byte(address)}}

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

				if len(label) > 0 {
					checkCodeLabel(label, codeLabels)

					// add the label to our table
					address := len(code)
					if address > 255 {
						vputils.CheckAndExit(errors.New("Exceeded code label table size"))
					}
					codeLabels[label] = Address{[]byte{byte(address)}}
				}

				target, tokens := first(tokens)

				// check there are no more tokens
				if len(tokens) > 0 {
					vputils.CheckAndExit(errors.New("Extra tokens on line"))
				}

				// decode the instruction
				instruction := getInstruction(opcode, targetSize, target, opcodeDefs, dataLabels, codeLabels)

				code = append(code, instruction...)
			}
		}
	}
	fmt.Println("\t\tENDSEGMENT")
	fmt.Println()

	return data, dataLabels, codeLabels
}

func generateCode(source []string, opcodeDefs map[string]opcodeDefinition, dataLabels LabelTable, codeLabels LabelTable) []byte {
	fmt.Println("\t\tCODE")

	code := []byte{}

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
				instruction := getInstruction(opcode, targetSize, target, opcodeDefs, dataLabels, codeLabels)

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

func write(properties []vputils.NameValue, code []byte, exports []vputils.NameValue, data []byte, filename string, codeAddressWidth int, dataAddressWidth int) {
	f, err := os.Create(filename)
	vputils.CheckAndPanic(err)

	defer f.Close()

	vputils.WriteString(f, "module")

	vputils.WriteTextTable("properties", properties, f)
	vputils.WriteTextTable("exports", exports, f)
	vputils.WriteBinaryBlock("code", code, f, codeAddressWidth)
	vputils.WriteBinaryBlock("data", data, f, dataAddressWidth)

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

	empty_opcodes := make(opcodeAddresses)

	opcodeDefs["EXIT"] = opcodeDefinition{0x00, empty_opcodes, false}
	opcodeDefs["OUT"] = opcodeDefinition{0x08, empty_opcodes, false}
	opcodeDefs["JUMP"] = opcodeDefinition{0x90, empty_opcodes, true}
	opcodeDefs["JZ"] = opcodeDefinition{0x92, empty_opcodes, true}

	push_opcodes := make(opcodeAddresses)
	push_opcodes["B"] = []byte{0x40, 0x41, 0x42, 0x0F}
	opcodeDefs["PUSH"] = opcodeDefinition{0x0F, push_opcodes, false}

	pop_opcodes := make(opcodeAddresses)
	pop_opcodes["B"] = []byte{0x0F, 0x51, 0x52, 0x0F}
	opcodeDefs["POP"] = opcodeDefinition{0x0F, pop_opcodes, false}

	flags_opcodes := make(opcodeAddresses)
	flags_opcodes["B"] = []byte{0x0F, 0x11, 0x12, 0x13}
	opcodeDefs["FLAGS"] = opcodeDefinition{0x0F, flags_opcodes, false}

	inc_opcodes := make(opcodeAddresses)
	inc_opcodes["B"] = []byte{0x0F, 0x21, 0x22, 0x23}
	opcodeDefs["INC"] = opcodeDefinition{0x0F, inc_opcodes, false}

	return opcodeDefs
}

func makeExports(codeLabels LabelTable) []vputils.NameValue {
	exports := []vputils.NameValue{}

	for label, address := range codeLabels {
		if vputils.IsUpper(label[0]) {
			s := address.to_s()
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

	sourceFile := args[0]

	moduleFile := ""

	if len(args) > 1 {
		moduleFile = args[1]
	}

	opcodeDefs := makeOpcodeDefinitions()
	instructionSetVersion := "1"

	codeAddressWidth := 1
	dataAddressWidth := 1

	properties := makeProperties(instructionSetVersion, codeAddressWidth, dataAddressWidth)

	source := vputils.ReadFile(sourceFile)

	data, dataLabels, codeLabels := generateData(source, opcodeDefs)

	exports := makeExports(codeLabels)

	code := generateCode(source, opcodeDefs, dataLabels, codeLabels)

	// if output specified, write module file
	if len(moduleFile) > 0 {
		write(properties, code, exports, data, moduleFile, codeAddressWidth, dataAddressWidth)
	}
}

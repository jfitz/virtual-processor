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

func buildInstruction(opcodes []byte, target string, dataLabels map[string]byte) ([]byte, error) {
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
		value := dataLabels[target]
		return []byte{opcode, value}, nil
	}

	if vputils.IsDirectAddress(target) {
		opcode := opcodes[1]
		value := dataLabels[target[1:]]
		return []byte{opcode, value}, nil
	}

	if vputils.IsIndirectAddress(target) {
		opcode := opcodes[2]
		value := dataLabels[target[2:]]
		return []byte{opcode, value}, nil
	}

	return nil, errors.New("Invalid opcode")
}

func decodeOpcode(text string, target string, codeLabels map[string]byte) ([]byte, []byte, error) {
	opcode := []byte{}
	opcodes := []byte{}

	switch text {
	case "EXIT":
		opcode = []byte{0x00}
	case "PUSH.B":
		opcodes = []byte{0x40, 0x41, 0x42, 0x0F}
	case "POP.B":
		opcodes = []byte{0x0F, 0x51, 0x52, 0x0F}
	case "OUT.B":
		opcode = []byte{0x08}
	case "FLAGS.B":
		opcodes = []byte{0x0F, 0x11, 0x12, 0x13}
	case "JUMP":
		address := codeLabels[target]
		opcode = []byte{0x90, address}
	case "JZ":
		address := codeLabels[target]
		opcode = []byte{0x92, address}
	case "INC.B":
		opcodes = []byte{0x0F, 0x21, 0x22, 0x23}
	default:
		return []byte{}, []byte{}, errors.New("Invalid opcode: '" + text + "' ")
	}

	return opcode, opcodes, nil
}

func getInstruction(text string, target string, dataLabels map[string]byte, codeLabels map[string]byte) []byte {
	instruction := []byte{}

	opcode, opcodes, err := decodeOpcode(text, target, codeLabels)
	vputils.CheckAndExit(err)

	if len(opcode) > 0 {
		// the instruction does not depend on target
		instruction = opcode
	}

	if len(opcodes) > 0 {
		// select instruction depends on target
		instr, err := buildInstruction(opcodes, target, dataLabels)
		vputils.CheckAndPanic(err)

		instruction = instr
	}

	return instruction
}

func checkDataLabel(label string, labels map[string]byte) {
	if label == "" {
		vputils.CheckAndExit(errors.New("Data declaration requires label"))
	}

	if _, ok := labels[label]; ok {
		vputils.CheckAndExit(errors.New("Duplicate label " + label))
	}
}

func checkCodeLabel(label string, labels map[string]byte) {
	if _, ok := labels[label]; ok {
		vputils.CheckAndExit(errors.New("Duplicate label " + label))
	}
}

func printLabels(labels map[string]byte) {
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

func generateData(source []string) ([]byte, map[string]byte, map[string]byte) {
	fmt.Println("\t\tDATA")

	code := []byte{}
	codeLabels := make(map[string]byte)
	data := []byte{}
	dataLabels := make(map[string]byte)

	for _, line := range source {
		// remove comment from line
		// remove trailing whitespace
		line = strings.TrimRight(line, " \t")
		// only lines with content
		if len(line) > 0 {
			tokens := vputils.Tokenize(line)
			label, tokens := first(tokens)
			opcode, tokens := first(tokens)

			// write the directive or instruction
			if isDirective(opcode) {
				checkDataLabel(label, dataLabels)

				// add the label to our table
				address := len(data)
				if address > 255 {
					vputils.CheckAndExit(errors.New("Exceeded data label table size"))
				}
				dataLabels[label] = byte(address)

				// write the label on a line by itself
				if len(label) > 0 {
					fmt.Printf("%s:\n", label)
				}

				values := []byte{}
				switch opcode {
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
					vputils.CheckAndExit(errors.New("Invalid directive " + opcode))
				}

				// print offset, directive, and contents
				location := len(data)

				if len(values) == 0 {
					fmt.Printf("%02X\t\t%s\n", location, opcode)
				} else {
					fmt.Printf("%02X\t\t%s\t\t% X\n", location, opcode, values)
					data = append(data, values...)
				}
			} else {
				// process instruction

				if len(label) > 0 {
					checkCodeLabel(label, codeLabels)

					// add the label to our table
					address := len(code)
					if address > 255 {
						vputils.CheckAndExit(errors.New("Exceeded code label table size"))
					}
					codeLabels[label] = byte(address)
				}

				target, tokens := first(tokens)

				// check there are no more tokens
				if len(tokens) > 0 {
					vputils.CheckAndExit(errors.New("Extra tokens on line"))
				}

				// decode the instruction
				instruction := getInstruction(opcode, target, dataLabels, codeLabels)

				code = append(code, instruction...)
			}
		}
	}
	fmt.Println("\t\tENDSEGMENT")
	fmt.Println()

	return data, dataLabels, codeLabels
}

func generateCode(source []string, dataLabels map[string]byte, codeLabels map[string]byte) []byte {
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
			opcode, tokens := first(tokens)

			// write the directive or instruction
			if !isDirective(opcode) {
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
				instruction := getInstruction(opcode, target, dataLabels, codeLabels)

				location := len(code)

				fmt.Printf("%02X\t% X\t%s\t%s\n", location, instruction, opcode, target)

				code = append(code, instruction...)
			}
		}
	}
	fmt.Println("\t\tENDSEGMENT")
	fmt.Println()

	return code
}

func write(properties []vputils.NameValue, code []byte, codeLabels map[string]byte, data []byte, filename string, codeWidth int, dataWidth int) {
	exports := []vputils.NameValue{}

	for k, v := range codeLabels {
		if vputils.IsUpper(k[0]) {
			s := strconv.Itoa(int(v))
			nv := vputils.NameValue{k, s}
			exports = append(exports, nv)
		}
	}

	fmt.Printf("Writing file %s...\n", filename)

	f, err := os.Create(filename)
	vputils.CheckAndPanic(err)

	defer f.Close()

	vputils.WriteString(f, "module")

	vputils.WriteTextTable("properties", properties, f)
	vputils.WriteTextTable("exports", exports, f)
	vputils.WriteBinaryBlock("code", code, f, codeWidth)
	vputils.WriteBinaryBlock("data", data, f, dataWidth)

	f.Sync()
}

func main() {
	args := os.Args[1:]

	if len(args) == 0 {
		fmt.Println("No source file specified")
		os.Exit(1)
	}

	sourceFile := args[0]

	if len(args) == 1 {
		fmt.Println("No output file specified")
		os.Exit(1)
	}

	moduleFile := args[1]

	codeAddressWidth := 1
	dataAddressWidth := 1

	properties := []vputils.NameValue{}

	caws := strconv.Itoa(codeAddressWidth)
	daws := strconv.Itoa(dataAddressWidth)

	properties = append(properties, vputils.NameValue{"STACK WIDTH", "1"})
	properties = append(properties, vputils.NameValue{"DATA WIDTH", "1"})
	properties = append(properties, vputils.NameValue{"ADDRESS WIDTH", "1"})
	properties = append(properties, vputils.NameValue{"CODE ADDRESS WIDTH", caws})
	properties = append(properties, vputils.NameValue{"DATA ADDRESS WIDTH", daws})
	properties = append(properties, vputils.NameValue{"CALL STACK SIZE", "1"})

	source := vputils.ReadFile(sourceFile)
	data, dataLabels, codeLabels := generateData(source)
	code := generateCode(source, dataLabels, codeLabels)

	write(properties, code, codeLabels, data, moduleFile, codeAddressWidth, dataAddressWidth)
}

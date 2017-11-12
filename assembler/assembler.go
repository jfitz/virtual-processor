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

	return false
}

func evaluateByte(expression string) byte {
	value, err := strconv.Atoi(expression)
	vputils.Check(err)

	byteValue := byte(value)

	return byteValue
}

func buildInstruction(opcodes []byte, target string, dataLabels map[string]byte) ([]byte, error) {
	instruction := []byte{}

	if vputils.IsDigit(target[0]) {
		opcode := opcodes[0]
		value := evaluateByte(target)
		instruction = []byte{opcode, value}
	}

	if vputils.IsAlpha(target[0]) {
		opcode := opcodes[1]
		value := dataLabels[target]
		instruction = []byte{opcode, value}
	}

	if len(instruction) == 0 {
		return instruction, errors.New("Invalid opcode")
	}

	return instruction, nil
}

func decodeOpcode(text string) (byte, []byte, error) {
	opcode := byte(0)
	opcodes := []byte{}

	switch text {
	case "EXIT":
		opcode = byte(0x00)
	case "PUSH.B":
		opcodes = []byte{0x40, 0x41}
	case "POP.B":
		opcode = byte(0x51)
	case "OUT.B":
		opcode = byte(0x08)
	default:
		return 0, []byte{}, errors.New("Invalid opcode: '" + text + "' ")
	}

	return opcode, opcodes, nil
}

func getInstruction(text string, target string, dataLabels map[string]byte) []byte {
	instruction := []byte{}

	opcode, opcodes, err := decodeOpcode(text)
	vputils.Check(err)

	if len(opcodes) == 0 {
		instruction = []byte{opcode}
	}

	if len(opcodes) > 0 {
		instr, err := buildInstruction(opcodes, target, dataLabels)
		vputils.Check(err)

		instruction = instr
	}

	return instruction
}

func checkDataLabel(label string, labels map[string]byte) {
	if label == "" {
		vputils.ShowErrorAndStop("Data declaration requires label")
	}

	if _, ok := labels[label]; ok {
		vputils.ShowErrorAndStop("Duplicate label " + label)
	}
}

func checkCodeLabel(label string, labels map[string]byte) {
	if _, ok := labels[label]; ok {
		vputils.ShowErrorAndStop("Duplicate label " + label)
	}
}

func printLabels(labels map[string]byte) {
	for k, v := range labels {
		fmt.Printf("%s\t%d\n", k, v)
	}
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
			tokens := vputils.Split(line)
			label, tokens := first(tokens)
			opcode, tokens := first(tokens)

			// write the directive or instruction
			if isDirective(opcode) {
				checkDataLabel(label, dataLabels)

				// add the label to our table
				address := len(dataLabels)
				if address > 255 {
					vputils.ShowErrorAndStop("Exceeded data label table size")
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
					value := evaluateByte(target)
					values = append(values, value)
				default:
					vputils.ShowErrorAndStop("Invalid directive")
				}

				// print offset, directive, and contents
				location := len(data)

				if len(values) == 0 {
					fmt.Printf("%02x\t\t%s\n", location, opcode)
				} else {
					fmt.Printf("%02x\t\t%s\t\t% X\n", location, opcode, values)
					data = append(data, values...)
				}
			} else {
				// process instruction

				if len(label) > 0 {
					checkCodeLabel(label, codeLabels)

					// add the label to our table
					address := len(codeLabels)
					if address > 255 {
						vputils.ShowErrorAndStop("Exceeded code label table size")
					}
					codeLabels[label] = byte(address)
				}

				target, tokens := first(tokens)

				// check there are no more tokens
				if len(tokens) > 0 {
					vputils.ShowErrorAndStop("Extra tokens on line")
				}

				// decode the instruction
				instruction := getInstruction(opcode, target, dataLabels)

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
			tokens := vputils.Split(line)
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
					vputils.ShowErrorAndStop("Extra tokens on line")
				}

				// decode the instruction
				instruction := getInstruction(opcode, target, dataLabels)

				location := len(code)

				fmt.Printf("%02x\t% X\t%s\t%s\n", location, instruction, opcode, target)

				code = append(code, instruction...)
			}
		}
	}
	fmt.Println("\t\tENDSEGMENT")
	fmt.Println()

	return code
}

func write(properties []vputils.NameValue, code []byte, data []byte, filename string, codeWidth int, dataWidth int) {
	fmt.Printf("Writing file %s...\n", filename)

	f, err := os.Create(filename)
	vputils.Check(err)

	defer f.Close()

	vputils.WriteString(f, "module")

	vputils.WriteTextTable("properties", properties, f)
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

	properties := []vputils.NameValue{}

	properties = append(properties, vputils.NameValue{"STACK WIDTH", "1"})
	properties = append(properties, vputils.NameValue{"DATA WIDTH", "1"})
	properties = append(properties, vputils.NameValue{"ADDRESS WIDTH", "1"})
	properties = append(properties, vputils.NameValue{"CODE ADDRESS WIDTH", "1"})
	properties = append(properties, vputils.NameValue{"DATA ADDRESS WIDTH", "1"})
	properties = append(properties, vputils.NameValue{"CALL STACK SIZE", "1"})

	source := vputils.ReadFile(sourceFile)
	data, dataLabels, codeLabels := generateData(source)
	code := generateCode(source, dataLabels, codeLabels)

	write(properties, code, data, moduleFile, 1, 1)
}

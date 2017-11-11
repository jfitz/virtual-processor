/*
package main of assembler
*/
package main

import (
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

func getInstruction(opcode string, target string) ([]byte, string) {
	instruction := []byte{}
	status := ""

	switch opcode {
	case "EXIT":
		instruction = []byte{0x00}
		status = ""
	case "PUSH.B":
		// PUSH B Value
		value := evaluateByte(target)
		instruction = []byte{0x40, value}
		status = ""
	case "POP.B":
		// POP B Address
		instruction = []byte{0x51}
		status = ""
	case "OUT.B":
		// OUT B
		instruction = []byte{0x13}
		status = ""
	default:
		status = "Invalid opcode: '" + opcode + "' "
	}

	return instruction, status
}

func checkDataLabel(label string, labels map[string]int) {
	if label == "" {
		vputils.ShowErrorAndStop("Data declaration requires label")
	}

	if _, ok := labels[label]; ok {
		vputils.ShowErrorAndStop("Duplicate label " + label)
	}
}

func checkCodeLabel(label string, labels map[string]int) {
	if _, ok := labels[label]; ok {
		vputils.ShowErrorAndStop("Duplicate label " + label)
	}
}

func printLabels(labels map[string]int) {
	for k, v := range labels {
		fmt.Printf("%s\t%d\n", k, v)
	}
}

func generateData(source []string) []byte {
	fmt.Println("Data segment:")

	data := []byte{}
	var data_labels map[string]int
	data_labels = make(map[string]int)

	for _, line := range source {
		// remove comment from line
		// remove trailing whitespace
		line = strings.TrimRight(line, " \t")
		// only lines with content
		if len(line) > 0 {
			tokens := vputils.Split(line)

			// first line must have op == 'START'

			label, tokens := first(tokens)

			opcode, tokens := first(tokens)

			// write the directive or instruction
			if isDirective(opcode) {
				checkDataLabel(label, data_labels)

				// add the label to our table
				data_labels[label] = len(data_labels)

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
				if len(values) == 0 {
					fmt.Printf("\t%s\n", opcode)
				} else {
					fmt.Printf("\t%s\t\t% X\n", opcode, values)
					data = append(data, values...)
				}
			}
		}
	}
	fmt.Println()

	fmt.Println("Data labels:")
	printLabels(data_labels)
	fmt.Println()

	return data
}

func generateCode(source []string) []byte {
	fmt.Println("Code segment:")

	code := []byte{}
	var code_labels map[string]int
	code_labels = make(map[string]int)

	for _, line := range source {
		// remove comment from line
		// remove trailing whitespace
		line = strings.TrimRight(line, " \t")
		// only lines with content
		if len(line) > 0 {
			tokens := vputils.Split(line)

			// first line must have op == 'START'

			label, tokens := first(tokens)

			opcode, tokens := first(tokens)

			// write the directive or instruction
			if !isDirective(opcode) {
				if len(label) > 0 {
					checkCodeLabel(label, code_labels)

					// add the label to our table
					code_labels[label] = len(code_labels)

					// write the label on a line by itself
					fmt.Printf("%s:\n", label)
				}

				target, _ := first(tokens)
				instruction, err := getInstruction(opcode, target)
				vputils.ShowErrorAndStop(err)

				fmt.Printf("\t%s\t%s\t% X\n", opcode, target, instruction)

				code = append(code, instruction...)
			}
		}
	}
	fmt.Println()

	fmt.Println("Code labels:")
	printLabels(code_labels)
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
	data := generateData(source)
	code := generateCode(source)

	write(properties, code, data, moduleFile, 1, 1)
}

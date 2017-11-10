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
	if s == "START" {
		return true
	}
	if s == "END" {
		return true
	}
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

func generateCode(source []string) ([]byte, []byte) {
	// for each line (while not done)

	code := []byte{}
	data := []byte{}
	for _, line := range source {
		// remove comment from line
		// remove trailing whitespace
		line = strings.TrimRight(line, " \t")
		// only lines with content
		if len(line) > 0 {
			tokens := vputils.Split(line)

			// first line must have op == 'START'

			label, tokens := first(tokens)

			// write the label on a line by itself
			if len(label) > 0 {
				fmt.Printf("\t%s:\n", label)
			}

			opcode, tokens := first(tokens)

			// write the directive or instruction
			if isDirective(opcode) {
				values := []byte{}
				switch opcode {
				case "START":
				case "END":
				case "BYTE":
					target, _ := first(tokens)
					value := evaluateByte(target)
					values = append(values, value)
				default:
					vputils.ShowErrorAndStop("Invalid directive")
				}
				fmt.Printf("\t\t%s\n", opcode)

				if len(values) > 0 {
					fmt.Printf("% X\n", values)
					data = append(data, values...)
				}
			} else {
				target, tokens := first(tokens)
				params, tokens := first(tokens)
				instruction, err := getInstruction(opcode, target)
				vputils.ShowErrorAndStop(err)

				fmt.Printf("\t\t%s\t%s\t%s\n", opcode, target, params)
				fmt.Printf("% X\n", instruction)

				code = append(code, instruction...)
			}
		}
	}

	return code, data
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
	code, data := generateCode(source)

	write(properties, code, data, moduleFile, 1, 1)
}

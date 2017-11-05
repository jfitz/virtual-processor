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

func parseLine(line string) (string, string, string) {
	trimmedLine := strings.TrimRight(line, " ")
	upcaseLine := strings.ToUpper(trimmedLine)
	parts := vputils.Split(upcaseLine, 3)
	label := ""
	opcode := ""
	args := ""
	if len(parts) > 0 {
		label = parts[0]
	}
	if len(parts) > 1 {
		opcode = parts[1]
	}
	if len(parts) > 2 {
		args = parts[2]
	}
	return label, opcode, args
}

func isDirective(s string) bool {
	if s == "START" {
		return true
	}
	if s == "END" {
		return true
	}
	if s == "DB" {
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

func getInstruction(opcode string, width string, target string) ([]byte, string) {
	instruction := []byte{}
	status := ""

	switch opcode {
	case "EXIT":
		instruction = []byte{0x00}
		status = ""
	case "PUSH":
		if width == "B" {
			// PUSH B V
			value := evaluateByte(target)
			instruction = []byte{0x40, value}
			status = ""
		}
	case "POP":
		// POP B A
		instruction = []byte{0x51}
		status = ""
	case "OUT":
		if width == "B" {
			// OUT B S
			instruction = []byte{0x13}
			status = ""
		}
	default:
		status = "Invalid opcode: '" + opcode + "' " + width
	}

	return instruction, status
}

func generateCode(source []string) []byte {
	// for each line (while not done)

	code := []byte{}
	for _, line := range source {
		// remove comment from line
		// remove trailing whitespace
		line = strings.TrimRight(line, " \t")
		// only lines with content
		if len(line) > 0 {
			// first line must have op == 'START'

			// split line into label, op, args
			label, opcode, rest := parseLine(line)

			// write the label on a line by itself
			if len(label) > 0 {
				fmt.Println(label + ":")
			}

			// write the directive or instruction
			if isDirective(opcode) {
				// if op == 'END' then add byte 0
				// when op == 'END' then flag as done
				if opcode == "DB" {
					code = append(code, 65)
				}
				fmt.Println("\t" + opcode)
			} else {
				width := ""
				target := ""
				params := ""
				args := vputils.Split(rest, 3)
				if len(args) > 0 {
					width = args[0]
				}
				if len(args) > 1 {
					target = args[1]
				}
				if len(args) > 2 {
					params = args[2]
				}
				instruction, err := getInstruction(opcode, width, target)
				vputils.ShowErrorAndStop(err)
				code = append(code, instruction...)
				fmt.Printf("% X\t%s\t%s\t%s\t%s\n", instruction, opcode, width, target, params)
			}
		}
	}

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
	code := generateCode(source)
	data := []byte{}

	write(properties, code, data, moduleFile, 1, 1)
}

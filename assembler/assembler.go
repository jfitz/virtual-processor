/*
package main of assembler
*/
package main

import (
	"fmt"
	"github.com/jfitz/virtual-processor/vputils"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
)

func check(e error) {
	if e != nil {
		panic(e)
	}
}

func showErrorAndStop(message string) {
	if message != "" {
		fmt.Println(message)
		os.Exit(1)
	}
}

func split(s string, max int) []string {
	parts := []string{}
	current := ""
	mode := true
	for _, c := range s {
		if (c == ' ' || c == '\t') && len(parts)+1 < max {
			if mode {
				parts = append(parts, current)
				current = ""
				mode = false
			}
		} else {
			current += string(c)
			mode = true
		}
	}
	if current != "" {
		parts = append(parts, current)
	}

	return parts
}

func readSource(sourceFile string) []string {
	fmt.Printf("Reading file %s...\n", sourceFile)
	b, err := ioutil.ReadFile(sourceFile)
	check(err)

	source := string(b)
	sourceLines := strings.Split(source, "\n")

	return sourceLines
}

func parseLine(line string) (string, string, string) {
	trimmedLine := strings.TrimRight(line, " ")
	upcaseLine := strings.ToUpper(trimmedLine)
	parts := split(upcaseLine, 3)
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
	check(err)
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
				args := split(rest, 3)
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
				showErrorAndStop(err)
				code = append(code, instruction...)
				fmt.Printf("% X\t%s\t%s\t%s\t%s\n", instruction, opcode, width, target, params)
			}
		}
	}

	return code
}

func writeBlockName(f *os.File, text string) {
	_, err := f.Write([]byte(text))
	check(err)

	zero_byte := []byte{0}

	_, err = f.Write(zero_byte)
	check(err)
}

// if length is greater than 65535 then error
func write2ByteLength(f *os.File, length int) {
	lHigh := byte(length & 0xff00 >> 8)
	lLow := byte(length & 0x00ff)
	lenBytes := []byte{lLow, lHigh}

	_, err := f.Write(lenBytes)
	check(err)
}

func writeBinaryBlock(name string, bytes []byte, f *os.File) {
	writeBlockName(f, name)
	write2ByteLength(f, len(bytes))

	_, err := f.Write(bytes)
	check(err)

	write2ByteLength(f, len(bytes))
}

func writeTextTable(name string, table []vputils.NameValue, f *os.File) {
	writeBlockName(f, name)

	stx_byte := []byte{0x02}
	etx_byte := []byte{0x03}
	fs_byte := []byte{0x1c}
	rs_byte := []byte{0x1e}

	// write STX
	_, err := f.Write(stx_byte)
	check(err)

	for _, nameValue := range table {
		name := []byte(nameValue.Name)
		value := []byte(nameValue.Value)

		// write name
		_, err = f.Write(name)
		check(err)
		// write FS
		_, err = f.Write(fs_byte)
		check(err)
		// write value
		_, err = f.Write(value)
		check(err)
		// write RS (0x1e)
		_, err = f.Write(rs_byte)
		check(err)
	}
	// write ETX
	_, err = f.Write(etx_byte)
	check(err)
}

func write(properties []vputils.NameValue, code []byte, data []byte, filename string) {
	fmt.Printf("Writing file %s...\n", filename)

	f, err := os.Create(filename)
	check(err)

	defer f.Close()

	writeBlockName(f, "module")

	writeTextTable("properties", properties, f)
	writeBinaryBlock("code", code, f)
	writeBinaryBlock("data", data, f)

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

	source := readSource(sourceFile)
	code := generateCode(source)
	data := []byte{}

	write(properties, code, data, moduleFile)
}

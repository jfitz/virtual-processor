/*
package of utilities for virtual-processor
*/
package vputils

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"
)

type NameValue struct {
	Name  string
	Value string
}

func Check(e error) {
	if e != nil {
		panic(e)
	}
}

func ShowErrorAndStop(message string) {
	if message != "" {
		fmt.Println(message)
		os.Exit(1)
	}
}

func Split(s string, max int) []string {
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

func Read2ByteLength(f *os.File) int {
	bytes := make([]byte, 2)
	_, err := f.Read(bytes)
	Check(err)

	length := int(bytes[1])<<8 + int(bytes[0])

	return length
}

// if length is greater than 65535 then error
func Write2ByteLength(f *os.File, length int) {
	lHigh := byte(length & 0xff00 >> 8)
	lLow := byte(length & 0x00ff)
	lenBytes := []byte{lLow, lHigh}

	_, err := f.Write(lenBytes)
	Check(err)
}

func ReadString(f *os.File) string {
	bytes := []byte{}
	one_byte := make([]byte, 1)
	one_byte[0] = 1
	for one_byte[0] != 0 {
		_, err := f.Read(one_byte)
		Check(err)
		if one_byte[0] != 0 {
			bytes = append(bytes, one_byte...)
		}
	}

	name := string(bytes)

	return name
}

func WriteString(f *os.File, text string) {
	_, err := f.Write([]byte(text))
	Check(err)

	zero_byte := []byte{0}

	_, err = f.Write(zero_byte)
	Check(err)
}

func ReadBinaryBlock(f *os.File) []byte {
	countBytes := Read2ByteLength(f)

	code := make([]byte, countBytes)
	_, err := f.Read(code)
	Check(err)

	checkCountBytes := Read2ByteLength(f)

	if checkCountBytes != countBytes {
		ShowErrorAndStop("Block count error")
	}

	return code
}

func WriteBinaryBlock(name string, bytes []byte, f *os.File) {
	WriteString(f, name)
	Write2ByteLength(f, len(bytes))

	_, err := f.Write(bytes)
	Check(err)

	Write2ByteLength(f, len(bytes))
}

func ReadTextTable(f *os.File) []NameValue {
	stx_byte := []byte{0x02}
	etx_byte := []byte{0x03}
	fs_byte := []byte{0x1c}
	rs_byte := []byte{0x1e}

	one_byte := make([]byte, 1)

	// read STX
	_, err := f.Read(one_byte)
	Check(err)

	if one_byte[0] != stx_byte[0] {
		ShowErrorAndStop("Did not find STX")
	}

	// read until ETX
	bytes := []byte{}
	one_byte[0] = 0
	for one_byte[0] != etx_byte[0] {
		_, err := f.Read(one_byte)
		Check(err)
		if one_byte[0] != etx_byte[0] {
			bytes = append(bytes, one_byte...)
		}
	}

	all_text := string(bytes)
	records := strings.Split(all_text, string(rs_byte))

	nameValues := []NameValue{}

	for _, record := range records {
		fields := strings.Split(record, string(fs_byte))
		if len(fields) == 2 {
			name := fields[0]
			value := fields[1]
			nameValue := NameValue{name, value}
			nameValues = append(nameValues, nameValue)
		}
	}

	return nameValues
}

func WriteTextTable(name string, table []NameValue, f *os.File) {
	WriteString(f, name)

	stx_byte := []byte{0x02}
	etx_byte := []byte{0x03}
	fs_byte := []byte{0x1c}
	rs_byte := []byte{0x1e}

	// write STX
	_, err := f.Write(stx_byte)
	Check(err)

	for _, nameValue := range table {
		name := []byte(nameValue.Name)
		value := []byte(nameValue.Value)

		// write name
		_, err = f.Write(name)
		Check(err)
		// write FS
		_, err = f.Write(fs_byte)
		Check(err)
		// write value
		_, err = f.Write(value)
		Check(err)
		// write RS (0x1e)
		_, err = f.Write(rs_byte)
		Check(err)
	}
	// write ETX
	_, err = f.Write(etx_byte)
	Check(err)
}

func ReadFile(sourceFile string) []string {
	fmt.Printf("Reading file %s...\n", sourceFile)
	b, err := ioutil.ReadFile(sourceFile)
	Check(err)

	source := string(b)
	sourceLines := strings.Split(source, "\n")

	return sourceLines
}

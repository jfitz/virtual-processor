/*
Package vputils for virtual-processor
*/
package vputils

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
)

// NameValue - hold a name and value pair
type NameValue struct {
	Name  string
	Value string
}

// CheckAndPanic - check for error
func CheckAndPanic(e error) {
	if e != nil {
		panic(e)
	}
}

// CheckAndExit - check for error
func CheckAndExit(e error) {
	if e != nil {
		fmt.Println(e.Error())
		os.Exit(1)
	}
}

// CheckPrintAndExit - check for error
func CheckPrintAndExit(e error, message string) {
	if e != nil {
		fmt.Println(e.Error() + " " + message)
		os.Exit(1)
	}
}

func checkWidth(width int) {
	if width != 1 && width != 2 {
		CheckAndExit(errors.New("Invalid width"))
	}
}

// IsSpace - is it a space
func IsSpace(c byte) bool {
	return c == ' ' || c == '\t'
}

// IsDoubleQuote - is it a quote
func IsDoubleQuote(c byte) bool {
	return c == '"'
}

// IsDigit - is it a digit
func IsDigit(c byte) bool {
	return c >= '0' && c <= '9'
}

// IsAlpha - is it an alphabetic
func IsAlpha(c byte) bool {
	return (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z')
}

// IsAlnum - is it alphabetic or digit
func IsAlnum(c byte) bool {
	return IsDigit(c) || IsAlpha(c) || c == '_'
}

// IsUpper - is it an uppercase alphabetic
func IsUpper(c byte) bool {
	return (c >= 'A' && c <= 'Z')
}

// IsLower - is it a lowercase alphabetic
func IsLower(c byte) bool {
	return (c >= 'a' && c <= 'z')
}

// IsText - anything that can be in a label or opcode
func IsText(c byte) bool {
	return IsAlnum(c) || c == '.' || c == ':'
}

// IsDirectAddress - is it a direct address
func IsDirectAddress(s string) bool {
	return len(s) >= 2 && s[0] == '@' && IsAlpha(s[1])
}

// IsIndirectAddress - is it an indirect address
func IsIndirectAddress(s string) bool {
	return len(s) >= 3 && s[0] == '@' && s[1] == '@' && IsAlpha(s[2])
}

// test for compatible character
func compatible(token string, c byte) bool {
	if token == "" {
		// empty token accepts anything
		return true
	}

	if IsSpace(token[0]) {
		// space token accepts spaces
		return IsSpace(c)
	}

	if IsDoubleQuote(token[0]) && len(token) == 1 {
		// quote by itself accepts anything
		return true
	}

	if IsDoubleQuote(token[0]) && len(token) > 1 && !IsDoubleQuote(token[len(token)-1]) {
		// quote with non-quote characters accepts anything
		return true
	}

	if IsDigit(token[0]) {
		// numeric token accepts digits
		return IsDigit(c)
	}

	if IsAlpha(token[0]) {
		// text token accepts alpha and digit and underscore and colon
		return IsText(c)
	}

	if token == "@" {
		return c == '@' || IsAlpha(c)
	}

	if token == "@@" {
		return IsAlpha(c)
	}

	// after checking for '@' and '@@', accept text (for labels)
	// we know there is at least one leading alpha
	if token[0] == '@' {
		return IsAlnum(c)
	}

	return false
}

// Tokenize - convert string into tokens
func Tokenize(s string) []string {
	parts := []string{}
	current := ""
	for _, c := range s {
		if compatible(current, byte(c)) {
			current += string(c)
		} else {
			// incompatible character requires a new token
			parts = append(parts, current)
			current = string(c)
		}
	}

	if current != "" {
		parts = append(parts, current)
	}

	return parts
}

func read1ByteInt(f *os.File) int {
	bytes := make([]byte, 1)
	_, err := f.Read(bytes)
	CheckAndPanic(err)

	value := int(bytes[0])

	return value
}

func read2ByteInt(f *os.File) int {
	bytes := make([]byte, 2)
	_, err := f.Read(bytes)
	CheckAndPanic(err)

	value := int(bytes[1])<<8 + int(bytes[0])

	return value
}

// if value is greater than 255 then error
func write1ByteInt(f *os.File, value int) {
	low := byte(value & 0x00ff)
	bytes := []byte{low}

	_, err := f.Write(bytes)
	CheckAndPanic(err)
}

// if value is greater than 65535 then error
func write2ByteInt(f *os.File, value int) {
	high := byte(value & 0xff00 >> 8)
	low := byte(value & 0x00ff)
	bytes := []byte{low, high}

	_, err := f.Write(bytes)
	CheckAndPanic(err)
}

// ReadString - read a string from a module file
func ReadString(f *os.File) string {
	bytes := []byte{}
	oneByte := make([]byte, 1)
	oneByte[0] = 1
	for oneByte[0] != 0 {
		_, err := f.Read(oneByte)
		CheckAndPanic(err)
		if oneByte[0] != 0 {
			bytes = append(bytes, oneByte...)
		}
	}

	name := string(bytes)

	return name
}

// WriteString - write a string to a module file
func WriteString(f *os.File, text string) {
	_, err := f.Write([]byte(text))
	CheckAndPanic(err)

	zeroByte := []byte{0}

	_, err = f.Write(zeroByte)
	CheckAndPanic(err)
}

// ReadBinaryBlock - read a binary block from a module file
func ReadBinaryBlock(f *os.File, width int) ([]byte, error) {
	checkWidth(width)

	countBytes := 0
	switch width {
	case 1:
		countBytes = read1ByteInt(f)
	case 2:
		countBytes = read2ByteInt(f)
	}

	code := make([]byte, countBytes)
	_, err := f.Read(code)
	CheckAndPanic(err)

	checkCountBytes := 0
	switch width {
	case 1:
		checkCountBytes = read1ByteInt(f)
	case 2:
		checkCountBytes = read2ByteInt(f)
	}

	if checkCountBytes != countBytes {
		return code, errors.New("Block count error")
	}

	return code, nil
}

// WriteBinaryBlock - write a binary block to a module file
func WriteBinaryBlock(name string, bytes []byte, f *os.File, width int) {
	checkWidth(width)

	WriteString(f, name)
	switch width {
	case 1:
		write1ByteInt(f, len(bytes))
	case 2:
		write2ByteInt(f, len(bytes))
	}

	_, err := f.Write(bytes)
	CheckAndPanic(err)

	switch width {
	case 1:
		write1ByteInt(f, len(bytes))
	case 2:
		write2ByteInt(f, len(bytes))
	}
}

// ReadTextTable - read a text table from a module file
func ReadTextTable(f *os.File) ([]NameValue, error) {
	stxByte := []byte{0x02}
	etxByte := []byte{0x03}
	fsByte := []byte{0x1c}
	rsByte := []byte{0x1e}

	oneByte := make([]byte, 1)

	nameValues := []NameValue{}

	// read STX
	_, err := f.Read(oneByte)
	if err != nil {
		return nameValues, errors.New("Could not read byte")
	}

	if oneByte[0] != stxByte[0] {
		return nameValues, errors.New("Did not find STX")
	}

	// read until ETX
	bytes := []byte{}
	oneByte[0] = 0
	for oneByte[0] != etxByte[0] {
		_, err := f.Read(oneByte)
		if err != nil {
			return nameValues, errors.New("Could not read byte")
		}

		if oneByte[0] != etxByte[0] {
			bytes = append(bytes, oneByte...)
		}
	}

	allText := string(bytes)
	records := strings.Split(allText, string(rsByte))

	for _, record := range records {
		fields := strings.Split(record, string(fsByte))
		if len(fields) == 2 {
			name := fields[0]
			value := fields[1]
			nameValue := NameValue{name, value}
			nameValues = append(nameValues, nameValue)
		}
	}

	return nameValues, nil
}

// WriteTextTable - write a text table to a module file
func WriteTextTable(name string, table []NameValue, f *os.File) error {
	WriteString(f, name)

	stxByte := []byte{0x02}
	etxByte := []byte{0x03}
	fsByte := []byte{0x1c}
	rsByte := []byte{0x1e}

	// write STX
	_, err := f.Write(stxByte)
	CheckAndPanic(err)

	for _, nameValue := range table {
		name := []byte(nameValue.Name)
		value := []byte(nameValue.Value)

		// write name
		_, err = f.Write(name)
		if err != nil {
			return errors.New("Failed to write name")
		}

		// write FS
		_, err = f.Write(fsByte)
		if err != nil {
			return errors.New("Failed to write FS")
		}

		// write value
		_, err = f.Write(value)
		if err != nil {
			return errors.New("Failed to write bytes")
		}

		// write RS (0x1e)
		_, err = f.Write(rsByte)
		if err != nil {
			return errors.New("Failed to write RS")
		}

	}
	// write ETX
	_, err = f.Write(etxByte)

	return err
}

// ReadFile - read a text file
func ReadFile(sourceFile string) ([]string, error) {
	b, err := ioutil.ReadFile(sourceFile)
	if err != nil {
		empty := make([]string, 0)
		return empty, errors.New("Failed to read file")
	}

	source := string(b)
	sourceLines := strings.Split(source, "\n")

	return sourceLines, nil
}

// Address --------------------------------
type Address struct {
	Value   int
	Size    int
	Maximum int
}

// MakeAddress - create an address
func MakeAddress(value int, size int, maximum int) (Address, error) {
	if value < 0 {
		return Address{0, 0, 0}, errors.New("Negative address")
	}

	if value > maximum {
		return Address{0, 0, 0}, errors.New("Address exceeds maximum")
	}

	return Address{value, size, maximum}, nil
}

// BytesToAddress - create an address
func BytesToAddress(bytes []byte, maximum int) (Address, error) {
	size := len(bytes)

	value := 0
	// convert bytes to int
	for i := size; i > 0; i-- {
		value *= 256
		value += int(bytes[i-1])
	}

	// built Address
	return Address{value, size, maximum}, nil
}

// Empty - is address empty
func (address Address) Empty() bool {
	return address.Size == 0
}

// ToString - convert to string
func (address Address) ToString() string {
	bytes := address.ToBytes()
	ss := []string{}
	for _, b := range bytes {
		s := fmt.Sprintf("%02X", b)
		ss = append(ss, s)
	}
	result := strings.Join(ss, " ")
	return result
}

// ToBytes - convert to array of bytes
func (address Address) ToBytes() []byte {
	v := address.Value
	bytes := []byte{}

	for i := 0; i < address.Size; i++ {
		b1 := byte(v & 0xff)
		bytes = append(bytes, b1)
		v = v / 256
	}

	return bytes
}

// AddByte - increase
func (address Address) Increment(i int) Address {
	a := address.Value + i
	return Address{a, address.Size, address.Maximum}
}

// Vector --------------------
type Vector []byte

// GetByte - get byte
func (v Vector) GetByte(address Address) (byte, error) {
	max := len(v) - 1
	offset := address.Value
	if offset < 0 || offset > max {
		offs := strconv.Itoa(offset)
		maxs := strconv.Itoa(max)
		return 0, errors.New("Index " + offs + " out of range [0.." + maxs + "]")
	}

	value := v[offset]
	return value, nil
}

// PutByte - put byte
func (v Vector) PutByte(address Address, value byte) error {
	max := len(v) - 1
	offset := address.Value
	if offset < 0 || offset > max {
		offs := strconv.Itoa(offset)
		maxs := strconv.Itoa(max)
		return errors.New("Index " + offs + " out of range [0.." + maxs + "]")
	}

	v[offset] = value

	return nil
}

// ----------------------------------------

// BoolStack ------------------------------
type BoolStack []bool

// Push - push bool value
func (stack BoolStack) Push(v bool) BoolStack {
	return append(stack, v)
}

// Top - get top value
func (stack BoolStack) Top() (bool, error) {
	if len(stack) < 1 {
		return false, errors.New("Stack underflow")
	}

	last := len(stack) - 1
	return stack[last], nil
}

// Pop - get top value
func (stack BoolStack) Pop() (bool, BoolStack, error) {
	if len(stack) < 1 {
		return false, stack, errors.New("Stack underflow")
	}

	last := len(stack) - 1
	return stack[last], stack[:last], nil
}

// ----------------------------------------

// ByteStack ------------------------------
type ByteStack []byte

// PushByte - push byte
func (stack ByteStack) PushByte(v byte) ByteStack {
	return append(stack, v)
}

func reverseBytes(bs []byte) []byte {
	last := len(bs) - 1

	for i := 0; i < len(bs)/2; i++ {
		bs[i], bs[last-i] = bs[last-i], bs[i]
	}

	return bs
}

// PushBytes - push bytes
func (stack ByteStack) PushBytes(vs []byte) ByteStack {
	bs := reverseBytes(vs)
	return append(stack, bs...)
}

// TopByte - get top byte
func (stack ByteStack) TopByte() (byte, error) {
	count := 1
	if len(stack) < count {
		return 0, errors.New("Stack underflow")
	}

	last := len(stack) - count
	return stack[last], nil
}

// PopByte - get top byte
func (stack ByteStack) PopByte(count int) ([]byte, ByteStack, error) {
	if len(stack) < count {
		return []byte{}, stack, errors.New("Stack underflow")
	}

	last := len(stack) - count
	return stack[last:], stack[:last], nil
}

// PushString - push a string
func (stack ByteStack) PushString(s string) ByteStack {
	bs := []byte(s)
	stack = stack.PushBytes(bs)
	b := byte(len(s))
	stack = stack.PushByte(b)

	return stack
}

// PopString - pop a string
func (stack ByteStack) PopString() (string, ByteStack) {
	// pop size of name
	counts, stack, err := stack.PopByte(1)
	CheckAndExit(err)
	count := int(counts[0])

	// pop bytes that make the string
	bytes := []byte{}
	s := ""
	for i := 0; i < count; i++ {
		bytes, stack, err = stack.PopByte(1)
		CheckAndExit(err)
		if bytes[0] != 0 {
			s += string(bytes[0])
		}
	}

	return s, stack
}

// ToByteString - convert to string of byte representation
func (stack ByteStack) ToByteString() string {
	s := ""

	if len(stack) > 0 {
		s = fmt.Sprintf("% 02X", stack)
	}

	return s
}

// ----------------------------------------

// AddressStack ---------------------------
type AddressStack []Address

// Push - push address
func (stack AddressStack) Push(address Address) AddressStack {
	return append(stack, address)
}

// Top - get top address
func (stack AddressStack) Top() (Address, error) {
	count := 1
	if len(stack) < count {
		return Address{0, 0, 0}, errors.New("Stack underflow")
	}

	last := len(stack) - count
	return stack[last], nil
}

// Pop - get top address
func (stack AddressStack) Pop() (AddressStack, error) {
	count := 1
	if len(stack) < count {
		return stack, errors.New("Stack underflow")
	}

	last := len(stack) - count
	return stack[:last], nil
}

// TopPop - get top address
func (stack AddressStack) TopPop() (Address, AddressStack, error) {
	count := 1
	if len(stack) < count {
		return Address{0, 0, 0}, stack, errors.New("Stack underflow")
	}

	last := len(stack) - count
	return stack[last], stack[:last], nil
}

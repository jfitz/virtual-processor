/*
package main of assembler
*/
package main

import (
	"errors"
	"fmt"
	"github.com/jfitz/virtual-processor/module"
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

func evaluateByte(expression string) []byte {
	value, err := strconv.Atoi(expression)
	vputils.CheckAndPanic(err)

	byteValue := byte(value)

	return []byte{byteValue}
}

func evaluateI16(expression string) []byte {
	value, err := strconv.Atoi(expression)
	vputils.CheckAndPanic(err)

	byteValue1 := byte(value & 0xff)
	byteValue2 := byte((value >> 8) & 0xff)

	return []byte{byteValue1, byteValue2}
}

type labelTable map[string]vputils.Address

type opcodeList map[string][]byte

func buildInstruction(opcodemap opcodeList, targetType string, target string, dataLabels labelTable) ([]byte, error) {
	opcodes, ok := opcodemap[targetType]

	if !ok {
		return nil, errors.New("Set '" + targetType + "' not found")
	}

	if len(target) == 0 {
		// stack
		opcode := opcodes[3]
		instruction := []byte{opcode}
		return instruction, nil
	}

	if vputils.IsDigit(target[0]) {
		// immediate value
		opcode := []byte{opcodes[0]}
		bytes := []byte{}
		switch targetType {
		case "B":
			bytes = evaluateByte(target)
		case "I16":
			bytes = evaluateI16(target)
		}
		instruction := append(opcode, bytes...)
		return instruction, nil
	}

	if vputils.IsAlpha(target[0]) {
		// immediate value
		opcode := []byte{opcodes[0]}
		address, ok := dataLabels[target]
		if !ok {
			err := errors.New("Undefined label '" + target + "'")
			vputils.CheckAndExit(err)
		}
		instruction := append(opcode, address.Bytes...)
		return instruction, nil
	}

	if vputils.IsDirectAddress(target) {
		// direct address
		opcode := []byte{opcodes[1]}
		address, ok := dataLabels[target[1:]]
		if !ok {
			err := errors.New("Undefined label '" + target + "'")
			vputils.CheckAndExit(err)
		}
		instruction := append(opcode, address.Bytes...)
		return instruction, nil
	}

	if vputils.IsIndirectAddress(target) {
		// indirect address
		opcode := []byte{opcodes[2]}
		address, ok := dataLabels[target[2:]]
		if !ok {
			err := errors.New("Undefined label '" + target + "'")
			vputils.CheckAndExit(err)
		}
		instruction := append(opcode, address.Bytes...)
		return instruction, nil
	}

	return nil, errors.New("Invalid opcode")
}

type opcodeDefinition struct {
	Opcode         byte
	AddressOpcodes opcodeList
}

func decodeOpcode(text string, instructionAddress vputils.Address, targetType string, target string, opcodeDefs map[string]opcodeDefinition, resolveAddress bool, codeLabels labelTable, dataLabels labelTable) ([]byte, error) {
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
		instruction, err = buildInstruction(addressOpcodes, targetType, target, dataLabels)
		vputils.CheckAndExit(err)
	}

	// for jump instructions, append the target address
	// TODO avoid hard-coded values
	firstByte := instruction[0]
	isJump := firstByte == 0xD0 || firstByte == 0xD1

	if isJump {
		address, ok := codeLabels[target]
		if !ok {
			if resolveAddress {
				err = errors.New("Undefined label '" + target + "'")
			} else {
				address, err = vputils.MakeAddress(0, 1, 0)
			}

			vputils.CheckAndExit(err)
		}

		// TODO: check address is within code address width
		instruction = append(instruction, address.Bytes...)
	}

	return instruction, nil
}

func getInstruction(text string, instructionAddress vputils.Address, targetType string, target string, opcodeDefs map[string]opcodeDefinition, resolveAddress bool, dataLabels labelTable, codeLabels labelTable) []byte {
	instruction, err := decodeOpcode(text, instructionAddress, targetType, target, opcodeDefs, resolveAddress, codeLabels, dataLabels)
	vputils.CheckAndExit(err)

	if len(instruction) == 0 {
		err := errors.New("Empty opcode")
		vputils.CheckAndExit(err)
	}

	return instruction
}

func checkDataLabel(label string, labels labelTable) {
	if label == "" {
		vputils.CheckAndExit(errors.New("Data declaration requires label"))
	}

	if _, ok := labels[label]; ok {
		vputils.CheckAndExit(errors.New("Duplicate label " + label))
	}
}

func checkCodeLabel(label string, labels labelTable) {
	if _, ok := labels[label]; ok {
		vputils.CheckAndExit(errors.New("Duplicate label " + label))
	}
}

func printLabels(labels labelTable) {
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

func encodeConditional(s string) ([]byte, error) {
	prefix := []byte{}

	if len(s) > 0 {
		parts := strings.Split(s, ".")

		for _, part := range parts {
			switch part {
			case "Z":
				prefix = append(prefix, 0xE0)
			case "NOT":
				prefix = append(prefix, 0xE8)
			default:
				return prefix, errors.New("Invalid conditional ")
			}
		}
	}

	return prefix, nil
}

func generateData(source []string, opcodeDefs map[string]opcodeDefinition) (vputils.Vector, labelTable, labelTable) {
	tabs := "\t\t\t"
	fmt.Println(tabs + "DATA")

	code := vputils.Vector{}
	codeLabels := make(labelTable)
	data := vputils.Vector{}
	dataLabels := make(labelTable)

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
				addressValue := len(data)
				if addressValue > 255 {
					vputils.CheckAndExit(errors.New("Exceeded data label table size"))
				}
				address, err := vputils.MakeAddress(addressValue, 1, len(data))
				vputils.CheckAndExit(err)
				dataLabels[label] = address

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
					values = append(values, value...)
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
					fmt.Printf("%02X%s%s\n", location, tabs, word)
				} else {
					fmt.Printf("%02X%s%s\t\t% X\n", location, tabs, word, values)
					data = append(data, values...)
				}
			} else {
				// process instruction
				opcode := word
				targetType := ""
				conditional := ""

				parts1 := strings.Split(word, ":")

				if len(parts1) > 1 {
					conditional = parts1[0]
					word = parts1[1]
				} else {
					word = parts1[0]
				}

				prefix, err := encodeConditional(conditional)
				vputils.CheckAndExit(err)

				parts2 := strings.Split(word, ".")

				if len(parts2) > 1 {
					opcode = parts2[0]
					targetType = parts2[1]
				} else {
					opcode = parts2[0]
				}

				address := len(code)
				// TODO: limit is based on address width
				if address > 255 {
					vputils.CheckAndExit(errors.New("Exceeded code label table size"))
				}

				instructionAddress, err := vputils.MakeAddress(address, 1, len(code))
				vputils.CheckAndExit(err)

				if len(label) > 0 {
					// add the label to our table
					checkCodeLabel(label, codeLabels)
					codeLabels[label] = instructionAddress
				}

				target, tokens := first(tokens)

				// check there are no more tokens
				if len(tokens) > 0 {
					fmt.Println(line)
					vputils.CheckAndExit(errors.New("Extra tokens on line"))
				}

				// decode the instruction
				instruction := getInstruction(opcode, instructionAddress, targetType, target, opcodeDefs, false, dataLabels, codeLabels)

				code = append(code, prefix...)
				code = append(code, instruction...)
			}
		}
	}

	fmt.Println(tabs + "ENDSEGMENT")
	fmt.Println()

	return data, dataLabels, codeLabels
}

func generateCode(source []string, opcodeDefs map[string]opcodeDefinition, dataLabels labelTable, codeLabels labelTable) vputils.Vector {
	tabs := "\t\t\t"
	fmt.Println(tabs + "CODE")

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
				mnemonic := ""
				targetType := ""
				conditional := ""

				parts1 := strings.Split(word, ":")

				if len(parts1) > 1 {
					conditional = parts1[0]
					mnemonic = parts1[1]
				} else {
					mnemonic = parts1[0]
				}

				prefix, err := encodeConditional(conditional)
				vputils.CheckAndExit(err)

				parts2 := strings.Split(mnemonic, ".")

				if len(parts2) > 1 {
					opcode = parts2[0]
					targetType = parts2[1]
				} else {
					opcode = parts2[0]
				}

				address := len(code)
				// TODO: limit is based on address width
				if address > 255 {
					vputils.CheckAndExit(errors.New("Exceeded code label table size"))
				}
				instructionAddress, err := vputils.MakeAddress(address, 1, len(code))
				vputils.CheckAndExit(err)

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
				instruction := getInstruction(opcode, instructionAddress, targetType, target, opcodeDefs, true, dataLabels, codeLabels)

				hexBytes := append(prefix, instruction...)
				location := len(code)
				instructionString := fmt.Sprintf("% X", hexBytes)
				// TODO: avoid hard-coded values for spacing
				wordTabCount := 2 - len(instructionString)/8
				wordTabs := ""
				for i := 0; i < wordTabCount; i++ {
					wordTabs += "\t"
				}

				fmt.Printf("%02X\t%s%s%s\t%s\n", location, instructionString, wordTabs, word, target)

				code = append(code, prefix...)
				code = append(code, instruction...)
			}
		}
	}
	fmt.Println(tabs + "ENDSEGMENT")
	fmt.Println()

	return code
}

func makeModuleProperties() []vputils.NameValue {
	properties := []vputils.NameValue{}

	properties = append(properties, vputils.NameValue{"CALL STACK SIZE", "1"})

	return properties
}

func makeCodeProperties(instructionSetVersion string, codeAddressWidth int, dataAddressWidth int) []vputils.NameValue {
	caws := strconv.Itoa(codeAddressWidth)
	daws := strconv.Itoa(dataAddressWidth)

	properties := []vputils.NameValue{}

	properties = append(properties, vputils.NameValue{"INSTRUCTION SET VERSION", instructionSetVersion})
	properties = append(properties, vputils.NameValue{"STACK WIDTH", "1"})
	properties = append(properties, vputils.NameValue{"DATA WIDTH", "1"})
	properties = append(properties, vputils.NameValue{"CODE ADDRESS WIDTH", caws})
	properties = append(properties, vputils.NameValue{"DATA ADDRESS WIDTH", daws})

	return properties
}

func makeDataProperties(dataAddressWidth int) []vputils.NameValue {
	daws := strconv.Itoa(dataAddressWidth)

	properties := []vputils.NameValue{}

	properties = append(properties, vputils.NameValue{"DATA WIDTH", "1"})
	properties = append(properties, vputils.NameValue{"DATA ADDRESS WIDTH", daws})

	return properties
}

func makeOpcodeDefinitions() map[string]opcodeDefinition {
	opcodeDefs := map[string]opcodeDefinition{}

	emptyOpcodes := make(opcodeList)

	opcodeDefs["NOP"] = opcodeDefinition{0x00, emptyOpcodes}
	opcodeDefs["EXIT"] = opcodeDefinition{0x04, emptyOpcodes}
	opcodeDefs["KCALL"] = opcodeDefinition{0x05, emptyOpcodes}
	opcodeDefs["OUT"] = opcodeDefinition{0x08, emptyOpcodes}

	opcodeDefs["JUMP"] = opcodeDefinition{0xD0, emptyOpcodes}

	opcodeDefs["CALL"] = opcodeDefinition{0xD1, emptyOpcodes}

	opcodeDefs["RET"] = opcodeDefinition{0xD2, emptyOpcodes}

	pushOpcodes := make(opcodeList)
	pushOpcodes["B"] = []byte{0x60, 0x61, 0x62, 0x0F}
	pushOpcodes["I16"] = []byte{0x64, 0x65, 0x66, 0x0F}
	pushOpcodes["STR"] = []byte{0x0F, 0x79, 0x7A, 0x0F}
	opcodeDefs["PUSH"] = opcodeDefinition{0x0F, pushOpcodes}

	popOpcodes := make(opcodeList)
	popOpcodes["B"] = []byte{0x0F, 0x81, 0x82, 0x83}
	opcodeDefs["POP"] = opcodeDefinition{0x0F, popOpcodes}

	flagsOpcodes := make(opcodeList)
	flagsOpcodes["B"] = []byte{0x10, 0x11, 0x12, 0x13}
	opcodeDefs["FLAGS"] = opcodeDefinition{0x0F, flagsOpcodes}

	incOpcodes := make(opcodeList)
	incOpcodes["B"] = []byte{0x0F, 0x21, 0x22, 0x23}
	opcodeDefs["INC"] = opcodeDefinition{0x0F, incOpcodes}

	decOpcodes := make(opcodeList)
	decOpcodes["B"] = []byte{0x0F, 0x31, 0x32, 0x33}
	opcodeDefs["DEC"] = opcodeDefinition{0x0F, decOpcodes}

	addOpcodes := make(opcodeList)
	addOpcodes["B"] = []byte{0x0F, 0x0F, 0x0F, 0xA0}
	opcodeDefs["ADD"] = opcodeDefinition{0x0F, addOpcodes}

	subOpcodes := make(opcodeList)
	subOpcodes["B"] = []byte{0x0F, 0x0F, 0x0F, 0xA1}
	opcodeDefs["SUB"] = opcodeDefinition{0x0F, subOpcodes}

	mulOpcodes := make(opcodeList)
	mulOpcodes["B"] = []byte{0x0F, 0x0F, 0x0F, 0xA2}
	opcodeDefs["MUL"] = opcodeDefinition{0x0F, mulOpcodes}

	divOpcodes := make(opcodeList)
	divOpcodes["B"] = []byte{0x0F, 0x0F, 0x0F, 0xA3}
	opcodeDefs["DIV"] = opcodeDefinition{0x0F, divOpcodes}

	andOpcodes := make(opcodeList)
	andOpcodes["B"] = []byte{0x0F, 0x0F, 0x0F, 0xC0}
	opcodeDefs["AND"] = opcodeDefinition{0x0F, andOpcodes}

	orOpcodes := make(opcodeList)
	orOpcodes["B"] = []byte{0x0F, 0x0F, 0x0F, 0xC1}
	opcodeDefs["OR"] = opcodeDefinition{0x0F, orOpcodes}

	cmpOpcodes := make(opcodeList)
	cmpOpcodes["B"] = []byte{0x0F, 0x0F, 0x0F, 0xC3}
	opcodeDefs["CMP"] = opcodeDefinition{0x0F, cmpOpcodes}

	return opcodeDefs
}

func makeExports(codeLabels labelTable) []vputils.NameValue {
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

func contains(list []string, s string) bool {
	for _, t := range list {
		if t == s {
			return true
		}
	}

	return false
}

func isSpace(s string) bool {
	if len(s) == 0 {
		return false
	}

	return s[0] == ' ' || s[0] == '\t'
}

func isString(s string) bool {
	if len(s) == 0 {
		return false
	}

	return s[0] == '"'
}

func isComment(s string) bool {
	if len(s) == 0 {
		return false
	}

	return s[0] == '#'
}

type tokenList []string

func (tokens tokenList) toString() string {
	s := ""

	for _, token := range tokens {
		s += token
		s += "_"
	}

	return s
}

func tokenizeLine(line string) tokenList {
	tokens := make([]string, 0)

	token := ""
	for _, c := range line {
		s := string(c)
		if len(token) == 0 || token[0] == '#' {
			// an empty token can accept any character
			// a comment can accept any character
			token += s
		} else {
			if s == "\"" {
				if isString(token) {
					token += s
					tokens = append(tokens, token)
					token = ""
				} else {
					tokens = append(tokens, token)
					token = s
				}
			} else {
				if (isSpace(token) == isSpace(s)) || isString(token) {
					token += s
				} else {
					tokens = append(tokens, token)
					token = s
				}
			}
		}
	}

	if len(token) > 0 {
		tokens = append(tokens, token)
	}

	return tokens
}

type lineAndTokens struct {
	Line   string
	Tokens tokenList
}

func tokenizeSource(source []string) []lineAndTokens {
	list := make([]lineAndTokens, 0)

	for _, line := range source {
		tokens := tokenizeLine(line)
		l := lineAndTokens{line, tokens}
		list = append(list, l)
	}

	return list
}

type tokenGroup struct {
	Labels       []string
	Nots         []string
	Conditionals []string
	Opcodes      []string
	Widths       []string
	CodeTargets  []string
	DataTargets  []string
	Values       []string
	Others       []string
}

type lineAndTokenGroup struct {
	Line   string
	Tokens tokenGroup
}

func isLabel(token string) bool {
	count := len(token)

	// must have some length
	if count == 0 {
		return false
	}

	// first must be alphabetic
	if !vputils.IsAlpha(token[0]) {
		return false
	}

	// last must be colon
	lastIndex := count - 1
	if token[lastIndex] != ':' {
		return false
	}

	// everything but last must be alpha-num or underscore
	text := token[0:lastIndex]

	for _, c := range text {
		b := byte(c)
		if !vputils.IsAlnum(b) {
			return false
		}
	}

	return true
}

func isValue(token string) bool {
	count := len(token)

	// must have some length
	if count == 0 {
		return false
	}

	// everything must be digit
	for _, c := range token {
		b := byte(c)
		if !vputils.IsDigit(b) {
			return false
		}
	}

	return true
}

func isCodeTarget(token string) bool {
	count := len(token)

	// must have some length
	if count == 0 {
		return false
	}

	// first must be alphabetic
	if !vputils.IsAlpha(token[0]) {
		return false
	}

	// everything must be alpha-num or underscore
	for _, c := range token {
		b := byte(c)
		if !vputils.IsAlnum(b) {
			return false
		}
	}

	return true
}

func isDataTarget(token string) bool {
	count := len(token)

	// must have some length
	if count == 0 {
		return false
	}

	// must have one or two '@' signs in front
	atCount := 0
	if token[0] == '@' {
		atCount++
	}

	if len(token) > 1 && token[1] == '@' {
		atCount++
	}

	if atCount == 0 {
		return false
	}

	text := token[atCount:len(token)]

	// must have some length after '@' signs
	if len(text) == 0 {
		return false
	}

	// first must be alphabetic
	if !vputils.IsAlpha(text[0]) {
		return false
	}

	// everything must be alpha-num or underscore
	for _, c := range text {
		b := byte(c)
		if !vputils.IsAlnum(b) {
			return false
		}
	}

	return true
}

func groupTokens(tokens tokenList) tokenGroup {
	groups := tokenGroup{}

	notList := []string{"NOT"}
	conditionalList := []string{"ZERO", "POSITIVE", "NEGATIVE"}
	widthList := []string{"BYTE", "I16", "I32", "I64", "F32", "F64", "STRING"}
	opcodeList := []string{"ADD", "SUB", "MUL", "DIV", "CMP", "PUSH", "POP", "EXIT", "KCALL", "OUT", "NOP", "JUMP", "CALL", "RET", "AND", "OR", "FLAGS", "INC", "DEC"}

	for _, token := range tokens {
		handled := false

		if isSpace(token) || isComment(token) {
			// do nothing, discard it
			handled = true
		}

		if isString(token) {
			groups.Values = append(groups.Values, token)
			handled = true
		}

		if contains(notList, token) {
			groups.Nots = append(groups.Nots, token)
			handled = true
		}

		if contains(widthList, token) {
			groups.Widths = append(groups.Widths, token)
			handled = true
		}

		if contains(conditionalList, token) {
			groups.Conditionals = append(groups.Conditionals, token)
			handled = true
		}

		if contains(opcodeList, token) {
			groups.Opcodes = append(groups.Opcodes, token)
			handled = true
		}

		if isLabel(token) {
			groups.Labels = append(groups.Labels, token)
			handled = true
		}

		if isValue(token) {
			groups.Values = append(groups.Values, token)
			handled = true
		}

		if !handled && isCodeTarget(token) {
			groups.CodeTargets = append(groups.CodeTargets, token)
			handled = true
		}

		if isDataTarget(token) {
			groups.DataTargets = append(groups.DataTargets, token)
			handled = true
		}

		if !handled {
			groups.Others = append(groups.Others, token)
		}
	}

	return groups
}

func group(list []lineAndTokens) []lineAndTokenGroup {
	groupList := make([]lineAndTokenGroup, 0)

	for _, tokens := range list {
		group := groupTokens(tokens.Tokens)
		l := lineAndTokenGroup{tokens.Line, group}
		groupList = append(groupList, l)
	}

	return groupList
}

func validateLine(lineAndTokens lineAndTokenGroup) bool {
	tokens := lineAndTokens.Tokens
	countLabels := len(tokens.Labels)
	countNots := len(tokens.Nots)
	countConditionals := len(tokens.Conditionals)
	countOpcodes := len(tokens.Opcodes)
	countWidths := len(tokens.Widths)
	countCodeTargets := len(tokens.CodeTargets)
	countDataTargets := len(tokens.DataTargets)
	countValues := len(tokens.Values)
	countOthers := len(tokens.Others)

	// any unrecognized token is invalid
	if countOthers > 0 {
		return false
	}

	// a blank line is valid
	if countLabels == 0 && countNots == 0 && countConditionals == 0 &&
		countOpcodes == 0 && countWidths == 0 && countCodeTargets == 0 &&
		countDataTargets == 0 && countValues == 0 {
		return true
	}

	// a data declaration has a label, width, and value
	if countLabels == 1 && countNots == 0 && countConditionals == 0 &&
		countOpcodes == 0 && countWidths == 1 && countCodeTargets == 0 &&
		countDataTargets == 0 && countValues == 1 {
		return true
	}

	countTargets := countCodeTargets + countDataTargets + countValues

	// opcodes may have a label, may have a width, may have a value or target
	// may have a conditional and may have a NOT
	if countLabels < 2 && countNots < 2 && countConditionals < 2 &&
		countOpcodes == 1 && countWidths < 2 && countTargets < 2 {
		return true
	}

	return false
}

func validate(groupList []lineAndTokenGroup) bool {
	allValid := true

	for _, lineAndTokens := range groupList {
		if !validateLine(lineAndTokens) {
			allValid = false

			fmt.Println(lineAndTokens.Line)
			fmt.Println("Invalid line")
		}
	}

	return allValid
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

	moduleProperties := makeModuleProperties()

	linesAndTokens := tokenizeSource(source)
	groupsList := group(linesAndTokens)
	valid := validate(groupsList)

	if !valid {
		vputils.CheckAndExit(errors.New("Errors found"))
	}

	data, dataLabels, codeLabels := generateData(source, opcodeDefs)
	dataProperties := makeDataProperties(dataAddressWidth)
	dataPage := module.Page{dataProperties, data}

	exports := makeExports(codeLabels)

	code := generateCode(source, opcodeDefs, dataLabels, codeLabels)
	codeProperties := makeCodeProperties(instructionSetVersion, codeAddressWidth, dataAddressWidth)
	codePage := module.Page{codeProperties, code}

	mod := module.Module{
		Properties:       moduleProperties,
		CodePage:         codePage,
		Exports:          exports,
		DataPage:         dataPage,
		CodeAddressWidth: codeAddressWidth,
		DataAddressWidth: dataAddressWidth,
	}

	// if output specified, write module file
	if len(moduleFile) > 0 {
		mod.Write(moduleFile)
	}
}

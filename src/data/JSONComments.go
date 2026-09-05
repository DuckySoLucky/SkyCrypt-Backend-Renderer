package data

import "strings"

// StripJSONComments removes JavaScript-style comments while preserving quoted strings.
func StripJSONComments(input string) string {
	var output strings.Builder
	output.Grow(len(input))

	inString := false
	escaped := false
	for index := 0; index < len(input); index++ {
		character := input[index]

		if inString {
			output.WriteByte(character)
			if escaped {
				escaped = false
			} else if character == '\\' {
				escaped = true
			} else if character == '"' {
				inString = false
			}
			continue
		}

		if character == '"' {
			inString = true
			output.WriteByte(character)
			continue
		}

		if character == '/' && index+1 < len(input) && input[index+1] == '/' {
			index += 2
			for index < len(input) && input[index] != '\n' && input[index] != '\r' {
				index++
			}
			if index < len(input) {
				output.WriteByte(input[index])
			}
			continue
		}

		if character == '/' && index+1 < len(input) && input[index+1] == '*' {
			index += 2
			for index+1 < len(input) && !(input[index] == '*' && input[index+1] == '/') {
				index++
			}
			if index+1 < len(input) {
				index++
			}
			continue
		}

		output.WriteByte(character)
	}

	return output.String()
}
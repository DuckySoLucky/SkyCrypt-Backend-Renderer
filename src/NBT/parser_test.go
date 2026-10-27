package nbt

import (
	"bytes"
	"encoding/binary"
	"testing"
)

func TestDecodeMutf8(t *testing.T) {
	tests := []struct {
		name  string
		input []byte
		want  string
	}{
		{"ascii", []byte("Hyperion"), "Hyperion"},
		{"two byte below U+0100", []byte{0xC2, 0xA7, '6'}, "§6"},
		{"two byte above U+00FF", []byte{0xD0, 0x96}, "Ж"},
		{"encoded nul", []byte{0xC0, 0x80}, "\x00"},
		{"three byte", []byte{0xE2, 0x9C, 0xAA}, "✪"},
		{"surrogate pair", []byte{0xED, 0xA0, 0xBD, 0xED, 0xB8, 0x80}, "😀"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := decodeMutf8(test.input)
			if err != nil {
				t.Fatal(err)
			}
			if got != test.want {
				t.Fatalf("decodeMutf8(% X) = %q, want %q", test.input, got, test.want)
			}
		})
	}
}

func TestReadStringLongerThanInt16(t *testing.T) {
	long := bytes.Repeat([]byte("a"), 40000)
	var buffer bytes.Buffer
	_ = binary.Write(&buffer, binary.BigEndian, uint16(len(long)))
	buffer.Write(long)
	buffer.WriteString("after")

	reader := &NbtBinaryReader{stream: &buffer}
	got, err := reader.ReadString()
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != len(long) {
		t.Fatalf("read %d bytes, want %d", len(got), len(long))
	}
	if rest := buffer.String(); rest != "after" {
		t.Fatalf("reader left %q in the stream, want %q", rest, "after")
	}
}

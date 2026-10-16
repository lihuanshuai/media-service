//go:build windows

package main

import (
	"os"
	"strings"
	"testing"
)

func TestBuildAutostartTaskXML(t *testing.T) {
	xml := buildAutostartTaskXML(`C:\Program Files\Media Service\App & "Name".exe`, "S-1-5-21-123-456")

	for _, want := range []string{
		`<UserId>S-1-5-21-123-456</UserId>`,
		`<RunLevel>HighestAvailable</RunLevel>`,
		`<LogonType>InteractiveToken</LogonType>`,
		`<Command>C:\Program Files\Media Service\App &amp; &quot;Name&quot;.exe</Command>`,
		`<MultipleInstancesPolicy>IgnoreNew</MultipleInstancesPolicy>`,
	} {
		if !strings.Contains(xml, want) {
			t.Errorf("XML 缺少 %q\n%s", want, xml)
		}
	}
}

func TestWriteUTF16LE(t *testing.T) {
	check := func(name, s string, want []byte) {
		t.Helper()
		f, err := os.CreateTemp("", "utf16-*.bin")
		if err != nil {
			t.Fatal(err)
		}
		defer os.Remove(f.Name())
		defer f.Close()

		if err := writeUTF16LE(f, s); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if _, err := f.Seek(0, 0); err != nil {
			t.Fatal(err)
		}
		data := make([]byte, len(want))
		if _, err := f.Read(data); err != nil {
			t.Fatal(err)
		}
		for i := range want {
			if data[i] != want[i] {
				t.Errorf("%s: 字节 %d = %#x, want %#x（完整输出 % x）", name, i, data[i], want[i], data)
			}
		}
	}

	// BOM + "A"(0x0041)
	check("ASCII", "A", []byte{0xFF, 0xFE, 0x41, 0x00})
	// BOM + 汉字"服"(0x670D)
	check("BMP", "服", []byte{0xFF, 0xFE, 0x0D, 0x67})
	// BOM + 代理对 "😀"(U+1F600 → D83D DE00)
	check("surrogate", "😀", []byte{0xFF, 0xFE, 0x3D, 0xD8, 0x00, 0xDE})
}

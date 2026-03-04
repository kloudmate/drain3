package drain3

import (
	"testing"
)

func TestMaskingIPAddress(t *testing.T) {
	inst, err := NewMaskingInstruction(`\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}`, "IP")
	if err != nil {
		t.Fatal(err)
	}

	masker := NewLogMasker([]*MaskingInstruction{inst}, "<", ">")

	result := masker.Mask("connection from 192.168.1.1 to 10.0.0.1")
	expected := "connection from <IP> to <IP>"
	if result != expected {
		t.Fatalf("expected %q, got %q", expected, result)
	}
}

func TestMaskingNumber(t *testing.T) {
	inst, err := NewMaskingInstruction(`\b\d+\b`, "NUM")
	if err != nil {
		t.Fatal(err)
	}

	masker := NewLogMasker([]*MaskingInstruction{inst}, "<", ">")

	result := masker.Mask("error code 42 at line 100")
	expected := "error code <NUM> at line <NUM>"
	if result != expected {
		t.Fatalf("expected %q, got %q", expected, result)
	}
}

func TestMaskingMultipleInstructions(t *testing.T) {
	inst1, err := NewMaskingInstruction(`\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}`, "IP")
	if err != nil {
		t.Fatal(err)
	}
	inst2, err := NewMaskingInstruction(`\b\d+\b`, "NUM")
	if err != nil {
		t.Fatal(err)
	}

	masker := NewLogMasker([]*MaskingInstruction{inst1, inst2}, "<", ">")

	result := masker.Mask("from 192.168.1.1 port 22")
	expected := "from <IP> port <NUM>"
	if result != expected {
		t.Fatalf("expected %q, got %q", expected, result)
	}
}

func TestMaskingCustomPrefixSuffix(t *testing.T) {
	inst, err := NewMaskingInstruction(`\d+`, "NUM")
	if err != nil {
		t.Fatal(err)
	}

	masker := NewLogMasker([]*MaskingInstruction{inst}, "{{", "}}")

	result := masker.Mask("port 8080")
	expected := "port {{NUM}}"
	if result != expected {
		t.Fatalf("expected %q, got %q", expected, result)
	}
}

func TestMaskingNoMatch(t *testing.T) {
	inst, err := NewMaskingInstruction(`\d+`, "NUM")
	if err != nil {
		t.Fatal(err)
	}

	masker := NewLogMasker([]*MaskingInstruction{inst}, "<", ">")

	result := masker.Mask("hello world")
	if result != "hello world" {
		t.Fatalf("expected unchanged, got %q", result)
	}
}

func TestMaskingNilMasker(t *testing.T) {
	var masker *LogMasker
	result := masker.Mask("hello 123")
	if result != "hello 123" {
		t.Fatalf("expected unchanged, got %q", result)
	}
}

func TestMaskNames(t *testing.T) {
	inst1, _ := NewMaskingInstruction(`\d+`, "NUM")
	inst2, _ := NewMaskingInstruction(`\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}`, "IP")
	inst3, _ := NewMaskingInstruction(`\b[A-F0-9]+\b`, "HEX")

	masker := NewLogMasker([]*MaskingInstruction{inst1, inst2, inst3}, "<", ">")

	names := masker.MaskNames()
	if len(names) != 3 {
		t.Fatalf("expected 3 names, got %d", len(names))
	}
}

func TestInstructionsByMaskName(t *testing.T) {
	inst1, _ := NewMaskingInstruction(`\d+`, "NUM")
	inst2, _ := NewMaskingInstruction(`\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}`, "IP")
	inst3, _ := NewMaskingInstruction(`0x[0-9a-f]+`, "NUM")

	masker := NewLogMasker([]*MaskingInstruction{inst1, inst2, inst3}, "<", ">")

	numInstructions := masker.InstructionsByMaskName("NUM")
	if len(numInstructions) != 2 {
		t.Fatalf("expected 2 NUM instructions, got %d", len(numInstructions))
	}

	ipInstructions := masker.InstructionsByMaskName("IP")
	if len(ipInstructions) != 1 {
		t.Fatalf("expected 1 IP instruction, got %d", len(ipInstructions))
	}
}

package hash

import "testing"

func TestHashUsesMD5PrefixInLittleEndianOrder(t *testing.T) {
	const input = "abc"

	if got, want := Hash16(input), uint16(0x0190); got != want {
		t.Fatalf("Hash16(%q) = %#x, want %#x", input, got, want)
	}
	if got, want := Hash32(input), uint32(0x98500190); got != want {
		t.Fatalf("Hash32(%q) = %#x, want %#x", input, got, want)
	}
	if got, want := Hash64(input), uint64(0xb04fd23c98500190); got != want {
		t.Fatalf("Hash64(%q) = %#x, want %#x", input, got, want)
	}
}

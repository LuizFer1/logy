package update

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
)

// SHA256Hex returns the lowercase hex SHA-256 of data.
func SHA256Hex(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

// VerifySHA256 checks that data hashes to wantHex (case-insensitive).
func VerifySHA256(data []byte, wantHex string) error {
	got := SHA256Hex(data)
	if !strings.EqualFold(got, strings.TrimSpace(wantHex)) {
		return fmt.Errorf("sha256 mismatch: got %s want %s", got, wantHex)
	}
	return nil
}

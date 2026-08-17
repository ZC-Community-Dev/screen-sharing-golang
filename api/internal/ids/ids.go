package ids

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math/big"
	"regexp"
	"strings"
)

const Alphabet = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"

var publicID = regexp.MustCompile(`^[0-9A-Za-z]{8,}$`)

func ValidPublicID(id string) bool {
	return publicID.MatchString(id)
}

func Generate(salt string) (string, error) {
	if salt == "" {
		return "", fmt.Errorf("salt is required")
	}
	raw := make([]byte, 16)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	mac := hmac.New(sha256.New, []byte(salt))
	mac.Write(raw)
	enc := EncodeBase62(mac.Sum(nil))
	if len(enc) < 10 {
		enc = enc + strings.Repeat("0", 10-len(enc))
	}
	return enc[:10], nil
}

func NewPresenterToken() (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return EncodeBase62(raw), nil
}

func HashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func EncodeBase62(b []byte) string {
	n := new(big.Int).SetBytes(b)
	if n.Sign() == 0 {
		return "0"
	}
	base := big.NewInt(62)
	zero := big.NewInt(0)
	mod := new(big.Int)
	var out []byte
	for n.Cmp(zero) > 0 {
		n.DivMod(n, base, mod)
		out = append(out, Alphabet[mod.Int64()])
	}
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	return string(out)
}

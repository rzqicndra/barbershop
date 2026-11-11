package app

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"unicode/utf8"

	"golang.org/x/crypto/argon2"
)

const (
	argonMemory = 64 * 1024
	argonTime   = 3
	argonLanes  = 1
)

func randomHex(bytes int) string {
	buffer := make([]byte, bytes)
	if _, err := rand.Read(buffer); err != nil {
		panic("operating system random source unavailable")
	}
	return hex.EncodeToString(buffer)
}

func newID() string {
	buffer := make([]byte, 16)
	if _, err := rand.Read(buffer); err != nil {
		panic("operating system random source unavailable")
	}
	buffer[6] = (buffer[6] & 0x0f) | 0x40
	buffer[8] = (buffer[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", buffer[:4], buffer[4:6], buffer[6:8], buffer[8:10], buffer[10:])
}

func PasswordHash(password string) (string, error) {
	length := utf8.RuneCountInString(password)
	if length < 12 || length > 128 {
		return "", invalid("Kata sandi harus berisi 12 sampai 128 karakter.")
	}
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	hash := argon2.IDKey([]byte(password), salt, argonTime, argonMemory, argonLanes, 32)
	return fmt.Sprintf("$argon2id$v=19$m=%d,t=%d,p=%d$%s$%s",
		argonMemory, argonTime, argonLanes,
		base64.RawStdEncoding.EncodeToString(salt), base64.RawStdEncoding.EncodeToString(hash)), nil
}

func verifyPassword(encoded, password string) bool {
	if utf8.RuneCountInString(password) > 128 {
		return false
	}
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 || parts[1] != "argon2id" || parts[2] != "v=19" {
		return false
	}
	var memory, iterations uint32
	var lanes uint8
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &memory, &iterations, &lanes); err != nil {
		return false
	}
	if memory < 19456 || memory > 262144 || iterations < 2 || iterations > 6 || lanes < 1 || lanes > 4 {
		return false
	}
	salt, saltErr := base64.RawStdEncoding.DecodeString(parts[4])
	expected, hashErr := base64.RawStdEncoding.DecodeString(parts[5])
	if saltErr != nil || hashErr != nil || len(salt) < 16 || len(expected) != 32 {
		return false
	}
	actual := argon2.IDKey([]byte(password), salt, iterations, memory, lanes, 32)
	return subtle.ConstantTimeCompare(actual, expected) == 1
}

func (a *App) bookingToken(booking Booking) string {
	payload := base64.RawURLEncoding.EncodeToString([]byte(booking.ID + ":" + booking.tokenNonce))
	mac := hmac.New(sha256.New, a.tokenKey)
	mac.Write([]byte("gilir-booking-v1:" + payload))
	return payload + "." + base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func (a *App) parseBookingToken(token string) (string, string, bool) {
	parts := strings.Split(token, ".")
	if len(parts) != 2 || len(token) > 250 {
		return "", "", false
	}
	signature, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return "", "", false
	}
	mac := hmac.New(sha256.New, a.tokenKey)
	mac.Write([]byte("gilir-booking-v1:" + parts[0]))
	if !hmac.Equal(signature, mac.Sum(nil)) {
		return "", "", false
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return "", "", false
	}
	values := strings.Split(string(payload), ":")
	if len(values) != 2 || !validID(values[0]) || len(values[1]) != 32 {
		return "", "", false
	}
	return values[0], values[1], true
}

func validID(id string) bool {
	if len(id) != 36 || id[8] != '-' || id[13] != '-' || id[18] != '-' || id[23] != '-' {
		return false
	}
	_, err := hex.DecodeString(strings.ReplaceAll(id, "-", ""))
	return err == nil
}

func requestHash(body any) []byte {
	// Callers provide a fixed-format, validated request struct.
	encoded, err := json.Marshal(body)
	if err != nil {
		panic("booking request cannot be encoded")
	}
	sum := sha256.Sum256(encoded)
	return sum[:]
}

func bookingCode() string {
	// The database UNIQUE constraint is the final authority for this public reference.
	return "GL-" + strings.ToUpper(randomHex(8))
}

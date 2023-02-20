package main

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"math/bits"
	"net/http"
	"strconv"
	"strings"
	"unicode"
)

func GetTermsOfService() (string, error) {
	const server = "https://webcash.org"

	resp, err := http.Get(server + "/terms/text")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(body), nil
}

type Uint256 [32]byte

func (hash Uint256) String() string {
	// encode as hex
	return fmt.Sprintf("0x%x", hash[:])
}

func ApparentDifficulty(hash Uint256) uint8 {
	diff := 0
	for i := 0; i < 32; i++ {
		c := hash[i]
		if c == 0 {
			diff += 8
			continue
		}
		diff += bits.LeadingZeros8(c)
		break
	}
	return uint8(diff)
}

func CheckProofOfWork(hash Uint256, difficulty uint8) bool {
	for i := 0; i < int(difficulty/8); i++ {
		if hash[i] != 0 {
			return false
		}
	}
	if difficulty%8 != 0 {
		if hash[difficulty/8]>>(8-difficulty%8) != 0 {
			return false
		}
	}
	return true
}

type Amount uint64

func (amt Amount) String() string {
	integer := uint64(amt) / 1_000_000_00
	decimal := uint64(amt) % 1_000_000_00
	decimalString := strconv.FormatUint(decimal, 10)
	if decimalString != "0" {
		// Pad with leading zeros
		for i := len(decimalString); i < 8; i++ {
			decimalString = "0" + decimalString
		}
		// Remove trailing zeros
		for decimalString[len(decimalString)-1] == '0' {
			decimalString = decimalString[:len(decimalString)-1]
		}
		return fmt.Sprintf("%d.%s", integer, decimalString)
	}
	return fmt.Sprintf("%d", integer)
}

func (amt Amount) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf("\"%s\"", fmt.Sprint(amt)[1:])), nil
}

func (amt *Amount) UnmarshalJSON(data []byte) error {
	var inner string
	if err := json.Unmarshal(data, &inner); err != nil {
		// Must not be wrapped as a string
		inner = string(data)
	}
	if strings.ContainsRune(inner, '.') {
		parts := strings.Split(inner, ".")
		if len(parts) != 2 {
			return fmt.Errorf("invalid amount: %v", data)
		}
		for _, rune := range parts[1] {
			if !unicode.IsDigit(rune) {
				return fmt.Errorf("invalid amount: %v", data)
			}
		}
		for i := len(parts[1]); i < 8; i++ {
			parts[1] += "0"
		}
		integer, err := strconv.ParseInt(parts[0], 10, 63)
		if err != nil {
			return err
		}
		decimal, err := strconv.ParseInt(parts[1], 10, 63)
		if err != nil {
			return err
		}
		*amt = Amount(integer*1_000_000_00 + decimal)
	} else {
		integer, err := strconv.ParseInt(inner, 10, 63)
		if err != nil {
			return err
		}
		*amt = Amount(integer * 1_000_000_00)
	}
	return nil
}

type SecretWebcash struct {
	// The actual secret, typically a 64-character hex string but in principle
	// any Unicode string value.
	Secret string `json:"secret"`
	// The amount of Webcash held by the secret.
	Amount Amount `json:"amount"`
}

func (sk SecretWebcash) String() string {
	return fmt.Sprintf("e%v:secret:%s", sk.Amount, sk.Secret)
}

type PublicWebcash struct {
	// The public hash, a 32-byte SHA-256 hash of the secret string.
	Hash Uint256 `json:"hash"`
	// The amount of Webcash held by the secret.
	Amount Amount `json:"amount"`
}

func (pk PublicWebcash) String() string {
	return fmt.Sprintf("e%v:public:%v", pk.Amount, pk)
}

// FromSecret converts a SecretWebcash to a PublicWebcash.
func FromSecret(sk SecretWebcash) PublicWebcash {
	return PublicWebcash{
		Hash:   sha256.Sum256([]byte(sk.Secret)),
		Amount: sk.Amount,
	}
}

type ProtocolSettings struct {
	// The number of leading bits which must be zero for a work candidate to be
	// accepted by the server.
	Difficulty uint8 `json:"difficulty_target_bits"`
	// The ratio of initial issuance distributed to expected amount.
	Ratio float32 `json:"ratio"`
	// The amount the miner is allowed to claim.
	TotalReward Amount `json:"mining_amount"`
	// The amount which is surrendered to the server operator.
	ServerSubsidy Amount `json:"mining_subsidy_amount"`
	// The number of subsidy adjustment periods which have elapsed.
	Epoch uint16 `json:"epoch"`
}

func get_protocol_settings() (ProtocolSettings, error) {
	const server = "https://webcash.org"

	resp, err := http.Get(server + "/api/v1/target")
	if err != nil {
		return ProtocolSettings{}, err
	}
	defer resp.Body.Close()
	var settings ProtocolSettings
	err = json.NewDecoder(resp.Body).Decode(&settings)
	if err != nil {
		return ProtocolSettings{}, err
	}
	return settings, nil
}

func main() {
	terms, err := GetTermsOfService()
	if err != nil {
		panic(err)
	}
	fmt.Println(terms)
	settings, err := get_protocol_settings()
	if err != nil {
		panic(err)
	}
	fmt.Println(settings)
	data, err := json.Marshal(settings)
	if err != nil {
		panic(err)
	}
	fmt.Println(string(data))
}

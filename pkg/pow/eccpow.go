package pow

import (
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"math/big"

	"github.com/painhardcore/SageSlayerServer/pkg/network"
)

// Challenge represents a PoW challenge.
type Challenge struct {
	Curve      elliptic.Curve
	Qx, Qy     *big.Int
	Difficulty int
}

// GenerateChallenge creates a new PoW challenge with the specified difficulty.
func GenerateChallenge(difficulty int) (*Challenge, error) {
	if difficulty < 0 {
		return nil, errors.New("Negative difficulty is not allowed")
	}
	curve := elliptic.P256()
	// Generate a random scalar k0
	k0, err := rand.Int(rand.Reader, curve.Params().N)
	if err != nil {
		return nil, err
	}
	// Compute random point Q = k0 * G
	Qx, Qy := curve.ScalarBaseMult(k0.Bytes())
	return &Challenge{
		Curve:      curve,
		Qx:         Qx,
		Qy:         Qy,
		Difficulty: difficulty,
	}, nil
}

// VerifySolution verifies the client's solution to the PoW challenge.
func VerifySolution(challenge *Challenge, nonceBytes []byte) error {
	Qx, Qy := challenge.Qx, challenge.Qy

	// Compute hash
	hashInput := append(Qx.Bytes(), Qy.Bytes()...)
	hashInput = append(hashInput, nonceBytes...)
	hash := sha256.Sum256(hashInput)

	// Check difficulty
	if !HasLeadingZeroBits(hash[:], challenge.Difficulty) {
		return errors.New("invalid solution: hash does not meet difficulty")
	}

	return nil
}

// SolveChallenge solves the PoW challenge (client-side).
func SolveChallenge(challengeProto *network.Challenge) ([]byte, error) {
	Qx := new(big.Int).SetBytes(challengeProto.Qx)
	Qy := new(big.Int).SetBytes(challengeProto.Qy)
	difficulty := int(challengeProto.Difficulty)

	nonce := big.NewInt(0)
	maxNonce := new(big.Int).Lsh(big.NewInt(1), 64) // 64-bit nonce limit
	one := big.NewInt(1)

	hashInputBase := append(Qx.Bytes(), Qy.Bytes()...)

	for nonce.Cmp(maxNonce) < 0 {
		nonceBytes := nonce.Bytes()
		hashInput := append(hashInputBase, nonceBytes...)
		hash := sha256.Sum256(hashInput)

		if HasLeadingZeroBits(hash[:], difficulty) {
			// Found a valid solution
			return nonceBytes, nil
		}
		nonce.Add(nonce, one)
	}
	return nil, errors.New("no valid solution found")
}

// HasLeadingZeroBits checks if the hash has the required number of leading zero bits.
func HasLeadingZeroBits(hash []byte, bits int) bool {
	bitIndex := 0
	for _, b := range hash {
		for i := 7; i >= 0; i-- {
			if bitIndex >= bits {
				return true
			}
			if ((b >> i) & 1) != 0 {
				return false
			}
			bitIndex++
		}
	}
	return bitIndex >= bits
}

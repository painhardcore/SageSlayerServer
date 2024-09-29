package pow_test

import (
	"crypto/elliptic"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"math/big"
	"testing"

	"github.com/painhardcore/SageSlayerServer/pkg/network"
	"github.com/painhardcore/SageSlayerServer/pkg/pow"
)

func TestGenerateChallenge(t *testing.T) {
	// Test with a valid difficulty
	challenge, err := pow.GenerateChallenge(1)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if challenge == nil {
		t.Fatal("expected a valid challenge, got nil")
	}
	if challenge.Difficulty != 1 {
		t.Fatalf("expected difficulty 10, got: %d", challenge.Difficulty)
	}
	if challenge.Qx == nil || challenge.Qy == nil {
		t.Fatal("expected valid Qx and Qy values, got nil")
	}
	if challenge.Curve != elliptic.P256() {
		t.Fatal("expected P256 curve")
	}

	// Test invalid randomness generation
	_, err = pow.GenerateChallenge(-1) // Edge case: invalid difficulty
	if err == nil {
		t.Fatal("expected error for invalid difficulty")
	}
}

func TestVerifySolution(t *testing.T) {
	challenge, err := pow.GenerateChallenge(10)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// Test with a valid solution
	solution, err := solveChallengeHelper(challenge)
	if err != nil {
		t.Fatalf("expected valid solution, got: %v", err)
	}

	err = pow.VerifySolution(challenge, solution)
	if err != nil {
		t.Fatalf("expected solution to be valid, got: %v", err)
	}

	// Test with an invalid solution (nonce that doesn't solve the challenge)
	invalidSolution := make([]byte, len(solution))
	copy(invalidSolution, solution)
	invalidSolution[0] ^= 0xff // Flip some bits to invalidate the solution

	err = pow.VerifySolution(challenge, invalidSolution)
	if err == nil {
		t.Fatal("expected invalid solution error")
	}
}

func TestSolveChallenge(t *testing.T) {
	challenge, err := pow.GenerateChallenge(1)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// Convert challenge to network proto formatdd
	challengeProto := &network.Challenge{
		Qx:         challenge.Qx.Bytes(),
		Qy:         challenge.Qy.Bytes(),
		Difficulty: int32(challenge.Difficulty),
	}

	solution, err := pow.SolveChallenge(challengeProto)
	if err != nil {
		t.Fatalf("expected solution to be found, got error: %v", err)
	}

	// Verify the solution
	err = pow.VerifySolution(challenge, solution)
	if err != nil {
		t.Fatalf("expected solution to be valid, got: %v", err)
	}
}

func TestHasLeadingZeroBits(t *testing.T) {
	// Precomputed hash with 4 leading zero bits (hex: 0x0f...)
	precomputedHash, _ := hex.DecodeString("0f00000000000000000000000000000000000000000000000000000000000000")

	// Test for valid zero bits count
	if !pow.HasLeadingZeroBits(precomputedHash, 4) {
		t.Fatal("expected hash to have leading zero bits")
	}

	// Edge case: Asking for more zero bits than possible
	if pow.HasLeadingZeroBits(precomputedHash, 256) {
		t.Fatal("expected hash not to have 256 leading zero bits")
	}
}

func solveChallengeHelper(challenge *pow.Challenge) ([]byte, error) {
	nonce := big.NewInt(0)
	maxNonce := new(big.Int).Lsh(big.NewInt(1), 64)
	one := big.NewInt(1)

	hashInputBase := append(challenge.Qx.Bytes(), challenge.Qy.Bytes()...)

	for nonce.Cmp(maxNonce) < 0 {
		nonceBytes := nonce.Bytes()
		hashInput := append(hashInputBase, nonceBytes...)
		hash := sha256.Sum256(hashInput)

		if pow.HasLeadingZeroBits(hash[:], challenge.Difficulty) {
			return nonceBytes, nil
		}
		nonce.Add(nonce, one)
	}
	return nil, errors.New("no valid solution found")
}

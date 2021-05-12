package encryption

import (
	"encoding/hex"
	"fmt"
	"testing"
)

var data = "0chain.net rocks"
var expectedHash = "6cb51770083ba34e046bc6c953f9f05b64e16a0956d4e496758b97c9cf5687d5"

func TestHash(t *testing.T) {
	if Hash(data) != expectedHash {
		fmt.Printf("invalid hash\n")
	} else {
		fmt.Printf("hash successful\n")
	}
}

func TestGenerateHash(t *testing.T) {
	keys := []string{
		"255452b9f49ebb8c8b8fcec9f0bd8a4284e540be1286bd562578e7e59765e41a7aada04c9e2ad3e28f79aacb0f1be66715535a87983843fea81f23d8011e728b",
		"17dd0eb10c2567899e34f368e1da2744c9cd3beb2e3babaa4e3350d950bad91dd4ed03d751d49e1f6c1b25c6ec8d1cc8db4ca2da72683d2958fb23843cc6a480",
		"aa182e7f1aa1cfcb6cad1e2cbf707db43dbc0afe3437d7d6c657e79cca732122f02a8106891a78b3ebaa2a37ebd148b7ef48f5c0b1b3311094b7f15a1bd7de12",
		"6059d12b7796b101e61c38bb994ebb830280870af2e4ee6d7d7810d110b8e50a61ede17bb5bf31b7208c19f3b9ca6f525ba8e17c14b9f7c8347e84c8375b8d1f",
		"6129267141b9ded7f3c87236859a392a122b42cb1b86677ca9268cca40e0ff0e276430707f44773a17ab6b5a1ce384e7ddd18046d03fe88fd70ccce60e358b8b",
		"ee39f9fae6818245657f2434e6a9a985753f650c34b092f45325631d8f222818c7ad33d599be6313ee668b72c304a9370e89844a6d52dda851e85ff171aa1f87",
		"623b7ac4ac6af5759244fa38262dc4aee1ca6ab9512f31f86d9e560764828415323810a495496b771b7d44c8ecdca9cbd10d13ca99167c56823fb00940a2baa3",
		"ca15cd56bd219918ff36cbc998a0422436965e8dbc973bb940b5a37c5011161ed2dbbc4bb33ce3773e128d9b6913d50dcbf89a56140256e9cd01268ff1f3699b",
	}
	for _, key := range keys {
		data, _ := hex.DecodeString(key)
		fmt.Println(Hash(data))
	}
}

package main

import (
	"strings"
	"testing"
)

// Known-positives for the two checks this program runs before it publishes.
//
// Both were written after the landing page had already shipped a caption
// calling an invented report real, and neither had ever been seen to fail.

func TestCheckSampleIsRealCatchesAPlantedFabrication(t *testing.T) {
	for _, tc := range []struct {
		name string
		blob string
		want string
	}{
		{
			"an invented checkpoint id",
			`{"checkpoint":{"id":"sample000class","commit":"0063443aa1"}}`,
			"invented checkpoint id",
		},
		{
			"a commit of all zeros",
			`{"checkpoint":{"id":"b20f84567474","commit":"0000000000000000000000000000000000000000"}}`,
			"placeholder commit",
		},
		{
			"no commit at all",
			`{"checkpoint":{"id":"b20f84567474","commit":""}}`,
			"placeholder commit",
		},
	} {
		err := CheckSampleIsReal([]byte(tc.blob))
		if err == nil {
			t.Errorf("%s: accepted", tc.name)
			continue
		}
		if !strings.Contains(err.Error(), tc.want) {
			t.Errorf("%s: error was %q, want it to mention %q", tc.name, err, tc.want)
		}
	}
}

// A real sample must still pass, or the check would be refusing everything and
// the tests above would prove nothing.
func TestCheckSampleIsRealAcceptsARealReport(t *testing.T) {
	blob := `{"checkpoint":{"id":"b20f84567474","commit":"0063443aa1bd0f6c5f0a2d6f0f1c3a9b8e7d6c5b"}}`
	if err := CheckSampleIsReal([]byte(blob)); err != nil {
		t.Errorf("a real sample was refused: %v", err)
	}
}

// The publish-time scan. example.invalid is reserved by RFC 2606 so this
// address can never be a real one.
func TestCheckSampleIsCleanCatchesAPlantedAddress(t *testing.T) {
	blob := `{"session_said":"emailed planted.address@example.invalid about it"}`
	err := CheckSampleIsClean("site/sample/umbra.json", []byte(blob))
	if err == nil {
		t.Fatal("a sample carrying an address was accepted for publication")
	}
	if !strings.Contains(err.Error(), "an email address") {
		t.Errorf("error was %q, want it to name the category", err)
	}
	if !strings.Contains(err.Error(), "site/sample/umbra.json") {
		t.Errorf("error was %q, want it to name the file", err)
	}
}

func TestCheckSampleIsCleanCatchesAPlantedHomePath(t *testing.T) {
	blob := `{"commands_run":["entire graph snapshot --repo /home/plantedperson/work"]}`
	if err := CheckSampleIsClean("site/sample/umbra.json", []byte(blob)); err == nil {
		t.Fatal("a sample carrying a home path was accepted for publication")
	}
}

func TestCheckSampleIsCleanAcceptsAScrubbedReport(t *testing.T) {
	blob := `{"commands_run":["entire graph snapshot --repo <repo>"],"session_said":"ran the tests"}`
	if err := CheckSampleIsClean("site/sample/umbra.json", []byte(blob)); err != nil {
		t.Errorf("a scrubbed sample was refused: %v", err)
	}
}

package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestKnownMainPatchChangesExactlyNineBytes(t *testing.T) {
	input, err := os.ReadFile("../../frontend/s/sv/" + originalMain)
	if err != nil {
		t.Skip(err)
	}
	if fmt.Sprintf("%x", sha256.Sum256(input)) != originalMainSHA {
		t.Fatal("unexpected source version")
	}
	data, err := unpackSWF(input, true)
	if err != nil {
		t.Fatal(err)
	}
	before := bytes.Clone(data)
	counts, err := patchMain(data)
	if err != nil {
		t.Fatal(err)
	}
	for _, key := range requiredPatches {
		if counts[key] != 1 {
			t.Fatalf("patch %s count=%d", key, counts[key])
		}
	}
	changed := 0
	for i := range data {
		if data[i] != before[i] {
			changed++
		}
	}
	if changed != 9 || len(data) != len(before) {
		t.Fatalf("changed %d bytes; fixed-length patch expected 9", changed)
	}
	offset, err := frameOffset(data)
	if err != nil || binary.LittleEndian.Uint16(data[offset:]) != 30*256 {
		t.Fatal("target is not 30 FPS")
	}
	result, err := packSWF(data, true)
	if err != nil {
		t.Fatal(err)
	}
	roundTrip, err := unpackSWF(result, true)
	if err != nil || !bytes.Equal(roundTrip, data) {
		t.Fatalf("SWF round-trip failed: %v", err)
	}
	if _, err := patchMain(data); err == nil {
		t.Fatal("must reject reapplying patch to trial")
	}
}

func TestKnownLoaderRedirectLeavesOriginalReferencesUntouched(t *testing.T) {
	input, err := os.ReadFile("../../frontend/s/sv/GameLoaders.swf")
	if err != nil {
		t.Skip(err)
	}
	if fmt.Sprintf("%x", sha256.Sum256(input)) != originalLoaderSHA {
		t.Fatal("unexpected loader version")
	}
	data, err := unpackSWF(input, false)
	if err != nil {
		t.Fatal(err)
	}
	before := bytes.Clone(data)
	if err := patchLoader(data); err != nil {
		t.Fatal(err)
	}
	if bytes.Count(data, []byte(trialMain)) != 3 || bytes.Contains(data, []byte(originalMain)) || bytes.Count(before, []byte(originalMain)) != 3 {
		t.Fatal("loader redirection changed wrong references")
	}
	if len(data) != len(before) {
		t.Fatal("loader string lengths changed")
	}
}

func TestInstructionPatchRejectsMissingAndDuplicatePatterns(t *testing.T) {
	for _, data := range [][]byte{{}, {0x24, 24, 0x61, 0x24, 24, 0x61}} {
		before := bytes.Clone(data)
		if err := changeOperand(data, []byte{0x24}, 24, 30, []byte{0x61}, "test", map[string]int{}); err == nil {
			t.Fatal("ambiguous/missing pattern accepted")
		}
		if !bytes.Equal(before, data) {
			t.Fatal("rejected pattern changed code")
		}
	}
}

func TestWriteNewRefusesOverwrite(t *testing.T) {
	path := filepath.Join(t.TempDir(), "trial.swf")
	if err := writeNew(path, []byte("original")); err != nil {
		t.Fatal(err)
	}
	if err := writeNew(path, []byte("replacement")); err == nil {
		t.Fatal("overwrote existing asset")
	}
	data, err := os.ReadFile(path)
	if err != nil || string(data) != "original" {
		t.Fatal("existing output changed")
	}
}

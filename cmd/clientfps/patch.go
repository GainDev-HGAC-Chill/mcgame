// Only verified class/method instruction sequences are changed, with unique-match checks.
package main

import (
	"bytes"
	"fmt"
	"strings"
)

var requiredPatches = []string{"GLOBAL_FRAME_RATE", "GLOBAL_FRAME_RATE_24", "GLOBAL_FRAME_RATE_20", "udcr", "udcp", "cbom", "move-threshold", "follow-threshold"}

func integerBytes(n uint32) []byte {
	var out []byte
	for {
		b := byte(n & 127)
		n >>= 7
		if n != 0 {
			b |= 128
		}
		out = append(out, b)
		if n == 0 {
			return out
		}
	}
}

func changeOperand(code, prefix []byte, old, next byte, suffix []byte, label string, counts map[string]int) error {
	pattern := append(bytes.Clone(prefix), old)
	pattern = append(pattern, suffix...)
	count := bytes.Count(code, pattern)
	if count != 1 {
		return fmt.Errorf("%s: expected one verified instruction sequence, got %d", label, count)
	}
	offset := bytes.Index(code, pattern) + len(prefix)
	code[offset] = next
	counts[label]++
	return nil
}

func patchCode(owner string, code []byte, names, stringsPool []string, counts map[string]int) error {
	if owner == "com.qeedoo.game.predef.GamePredef:initializer" {
		for _, setting := range []struct {
			name string
			old  byte
		}{{"GLOBAL_FRAME_RATE", 24}, {"GLOBAL_FRAME_RATE_24", 24}, {"GLOBAL_FRAME_RATE_20", 20}} {
			found := false
			for i, name := range names {
				if name != setting.name {
					continue
				}
				found = true
				index := integerBytes(uint32(i))
				prefix := append([]byte{0x5e}, index...)
				prefix = append(prefix, 0x24)
				suffix := append([]byte{0x61}, index...)
				if err := changeOperand(code, prefix, setting.old, 30, suffix, setting.name, counts); err != nil {
					return err
				}
			}
			if !found {
				return fmt.Errorf("missing QName %s", setting.name)
			}
		}
	}
	if owner == "com.qeedoo.game.config.RPCConfig:initializer" {
		for _, key := range []string{"udcr", "udcp", "cbom"} {
			found := false
			for i, name := range names {
				if name != "RPC_DENY_MAX" {
					continue
				}
				for j, value := range stringsPool {
					if value != key {
						continue
					}
					found = true
					prefix := append([]byte{0x60}, integerBytes(uint32(i))...)
					prefix = append(prefix, 0x2c)
					prefix = append(prefix, integerBytes(uint32(j))...)
					prefix = append(prefix, 0x24)
					if err := changeOperand(code, prefix, 24, 30, []byte{0x61}, key, counts); err != nil {
						return err
					}
				}
			}
			if !found {
				return fmt.Errorf("missing RPC key %s", key)
			}
		}
	}
	if strings.HasPrefix(owner, "com.qeedoo.effects.EnterFrameMove:") && strings.HasSuffix(owner, "nextPoint") {
		return changeOperand(code, []byte{0x24}, 35, 30, []byte{0x0d}, "move-threshold", counts)
	}
	if strings.HasPrefix(owner, "com.qeedoo.ui.view.compGameStage.CreatureView:") && strings.HasSuffix(owner, ".followHandler") {
		return changeOperand(code, []byte{0x24}, 40, 30, []byte{0x0d}, "follow-threshold", counts)
	}
	return nil
}

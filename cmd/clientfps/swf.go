// Experimental fixed-length AVM2 patches preserve tag sizes, branches and original files.
package main

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"fmt"
	"io"
)

func unpackSWF(input []byte, encoded bool) ([]byte, error) {
	data := bytes.Clone(input)
	if encoded {
		for i := range data {
			data[i] ^= byte((i + 1) % 3)
		}
	}
	if len(data) < 9 || string(data[:3]) != "CWS" {
		return nil, fmt.Errorf("expected CWS SWF")
	}
	r, err := zlib.NewReader(bytes.NewReader(data[8:]))
	if err != nil {
		return nil, err
	}
	defer r.Close()
	body, err := io.ReadAll(io.LimitReader(r, 128<<20))
	if err != nil {
		return nil, err
	}
	if len(body)+8 != int(binary.LittleEndian.Uint32(data[4:8])) {
		return nil, fmt.Errorf("SWF length mismatch")
	}
	return append(data[:8], body...), nil
}

func packSWF(data []byte, encoded bool) ([]byte, error) {
	var out bytes.Buffer
	out.Write(data[:8])
	w := zlib.NewWriter(&out)
	if _, err := w.Write(data[8:]); err != nil {
		return nil, err
	}
	if err := w.Close(); err != nil {
		return nil, err
	}
	result := out.Bytes()
	if encoded {
		for i := range result {
			result[i] ^= byte((i + 1) % 3)
		}
	}
	return result, nil
}

func frameOffset(data []byte) (int, error) {
	if len(data) < 9 {
		return 0, fmt.Errorf("missing RECT")
	}
	nbits := int(data[8] >> 3)
	offset := 8 + (5+4*nbits+7)/8
	if offset+4 > len(data) {
		return 0, fmt.Errorf("truncated SWF header")
	}
	return offset, nil
}

func patchMain(data []byte) (map[string]int, error) {
	offset, err := frameOffset(data)
	if err != nil {
		return nil, err
	}
	if binary.LittleEndian.Uint16(data[offset:]) != 24*256 {
		return nil, fmt.Errorf("expected original 24 FPS header")
	}
	counts := make(map[string]int)
	for cursor := offset + 4; cursor+2 <= len(data); {
		head := binary.LittleEndian.Uint16(data[cursor:])
		cursor += 2
		size := int(head & 63)
		if size == 63 {
			if cursor+4 > len(data) {
				return nil, fmt.Errorf("truncated tag size")
			}
			size = int(binary.LittleEndian.Uint32(data[cursor:]))
			cursor += 4
		}
		if size < 0 || size > len(data)-cursor {
			return nil, fmt.Errorf("invalid tag size")
		}
		if head>>6 == 82 {
			tag := data[cursor : cursor+size]
			if len(tag) < 5 {
				return nil, fmt.Errorf("truncated DoABC")
			}
			nameEnd := bytes.IndexByte(tag[4:], 0)
			if nameEnd < 0 {
				return nil, fmt.Errorf("missing ABC name")
			}
			if err := patchABC(tag[5+nameEnd:], counts); err != nil {
				return nil, err
			}
		}
		cursor += size
		if head>>6 == 0 {
			break
		}
	}
	for _, key := range requiredPatches {
		if counts[key] != 1 {
			return nil, fmt.Errorf("patch %s matched %d times, expected 1", key, counts[key])
		}
	}
	binary.LittleEndian.PutUint16(data[offset:], 30*256)
	return counts, nil
}

func patchLoader(data []byte) error {
	old := []byte(originalMain)
	if count := bytes.Count(data, old); count != 3 {
		return fmt.Errorf("expected exactly three loader main-asset references, got %d", count)
	}
	patched := bytes.ReplaceAll(data, old, []byte(trialMain))
	copy(data, patched)
	return nil
}

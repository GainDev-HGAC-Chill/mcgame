// Build: go build -o bin/clientfps.exe ./cmd/clientfps
// Generates opt-in 30 FPS copies; never overwrites originals or existing output files.
package main

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
)

const originalMain = "6a41a5ba5c0006ab4e0c1321a2b05027"
const trialMain = "6a41a5ba5c0006ab4e0c1321a2b05030"
const originalMainSHA = "07da21616d5ac18e2354cd9f2f04ab126fab85726033e6e861a7499bd064cd38"
const originalLoaderSHA = "e0092d207d306299bff3611fb37750c6c550be9fed6f1252d34cd34d1a707689"

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	root := filepath.Join("frontend", "s", "sv")
	mainPath, loaderPath := filepath.Join(root, originalMain), filepath.Join(root, "GameLoaders.swf")
	for _, name := range []string{trialMain, "GameLoaders30.swf"} {
		if _, err := os.Stat(filepath.Join(root, name)); !os.IsNotExist(err) {
			return fmt.Errorf("output %s already exists or is inaccessible; refusing overwrite", name)
		}
	}
	original, err := os.ReadFile(mainPath)
	if err != nil {
		return err
	}
	loader, err := os.ReadFile(loaderPath)
	if err != nil {
		return err
	}
	if fmt.Sprintf("%x", sha256.Sum256(original)) != originalMainSHA || fmt.Sprintf("%x", sha256.Sum256(loader)) != originalLoaderSHA {
		return fmt.Errorf("source assets do not match the researched version; refusing patch")
	}
	mainSWF, err := unpackSWF(original, true)
	if err != nil {
		return err
	}
	loaderSWF, err := unpackSWF(loader, false)
	if err != nil {
		return err
	}
	counts, err := patchMain(mainSWF)
	if err != nil {
		return err
	}
	if err := patchLoader(loaderSWF); err != nil {
		return err
	}
	mainResult, err := packSWF(mainSWF, true)
	if err != nil {
		return err
	}
	loaderResult, err := packSWF(loaderSWF, false)
	if err != nil {
		return err
	}
	if err := writeNew(filepath.Join(root, trialMain), mainResult); err != nil {
		return err
	}
	if err := writeNew(filepath.Join(root, "GameLoaders30.swf"), loaderResult); err != nil {
		if cleanupErr := os.Remove(filepath.Join(root, trialMain)); cleanupErr != nil {
			return fmt.Errorf("loader write failed: %v; cleanup of newly created trial main failed: %w", err, cleanupErr)
		}
		return err
	}
	for i, path := range []string{mainPath, loaderPath} {
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		before := original
		if i == 1 {
			before = loader
		}
		if sha256.Sum256(data) != sha256.Sum256(before) {
			return fmt.Errorf("original asset changed during generation: %s", path)
		}
		fmt.Printf("Original unchanged: %s SHA256=%x\n", path, sha256.Sum256(data))
	}
	fmt.Println("Verified patches:", counts)
	fmt.Println("Experimental target 30 FPS; animation pacing and live server load are NOT TESTED.")
	return nil
}

func writeNew(path string, data []byte) (err error) {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			if cleanupErr := os.Remove(path); cleanupErr != nil {
				err = fmt.Errorf("write failed: %v; cleanup of newly created file failed: %w", err, cleanupErr)
			}
		}
	}()
	if _, err := f.Write(data); err != nil {
		closeErr := f.Close()
		if closeErr != nil {
			return fmt.Errorf("write failed: %v; close failed: %w", err, closeErr)
		}
		return err
	}
	return f.Close()
}

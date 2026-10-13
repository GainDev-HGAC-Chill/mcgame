// game setup creates secrets only for a new deployment, never replacing existing state.
package main

import (
	"fmt"
	"os"
	"path/filepath"
)

func (s *stack) setup() error {
	if s.hasDatabase() {
		return fmt.Errorf("existing database: setup refuses to replace secrets or delete data; use game up")
	}
	if _, err := os.Stat(s.envPath); err == nil {
		return fmt.Errorf("docker/.env already exists: setup refuses to replace secrets; use game up")
	} else if !os.IsNotExist(err) {
		return err
	}

	env, err := loadEnv(filepath.Join(s.dir, ".env.example"))
	if err != nil {
		return err
	}
	env.path = s.envPath
	secrets, err := legacySecrets()
	if err != nil {
		return err
	}
	keys, err := authKeys(secrets["JWT_SECRET"])
	if err != nil {
		return err
	}
	for _, group := range []map[string]string{secrets, keys} {
		for key, value := range group {
			env.set(key, value)
		}
	}
	if err := env.save(); err != nil {
		return err
	}
	if err := os.Chmod(s.envPath, 0o600); err != nil {
		return err
	}
	if err := enableComposeJWKS(s.composePath); err != nil {
		fmt.Fprintln(os.Stderr, "warning:", err)
	}
	fmt.Println("===> docker/.env is ready; start the stack with: game up")
	return nil
}

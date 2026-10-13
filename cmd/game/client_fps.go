// The opt-in FPS trial is separate from the normal launcher and never restarts services.
package main

import (
	"fmt"
	"os"
	"path/filepath"
)

func client30URL(host string) string {
	return "http://" + host + ":8888/s/sv/GameLoaders30.swf?isExpand=true"
}

func (s *stack) openClient30() error {
	for _, name := range []string{"GameLoaders30.swf", "6a41a5ba5c0006ab4e0c1321a2b05030"} {
		if _, err := os.Stat(filepath.Join(s.root, "frontend", "s", "sv", name)); err != nil {
			return fmt.Errorf("30 FPS trial asset missing; build/run bin/clientfps.exe first: %w", err)
		}
	}
	fmt.Println("Experimental 30 FPS client. Close the normal projector before login.")
	fmt.Println("Animation timing and live server load are not yet verified; revert using MCGAME.bat option 2.")
	return s.launchClient(client30URL(s.clientHost()))
}

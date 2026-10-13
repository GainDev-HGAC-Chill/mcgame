package main

import "testing"

func TestTrialClientURLDoesNotChangeDefaultClient(t *testing.T) {
	for _, host := range []string{"127.0.0.1", "192.168.2.9"} {
		if got := client30URL(host); got != "http://"+host+":8888/s/sv/GameLoaders30.swf?isExpand=true" {
			t.Fatal(got)
		}
		if got := clientURL(host); got != "http://"+host+":8888/s/sv/GameLoaders.swf?isExpand=true" {
			t.Fatal(got)
		}
	}
}

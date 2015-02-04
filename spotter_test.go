package main

import (
	"testing"
	"encoding/json"
	"os"
)

func TestGetEvents(t *testing.T) {
	container := getContainerFromFile("test/container.json")
	hookSource := "test_123_hash_xyz:start,restart:pipework:eth0:{{.ID}}:192.168.242.1/24"
	hooks := getHooks(hookSource)
	events := GetEvents(hooks, &container)

	if (events == nil) {
		t.Error("Container not found by name", container.Name, "hooks: ", hookSource)
	}

	hookSource = "LIBVIRT_SERVICE_PORT=16509:start,restart:pipework:eth0:{{.ID}}:192.168.242.1/24"
	hooks = getHooks(hookSource)
	events = GetEvents(hooks, &container)

	if (events == nil) {
		t.Error("Container not found by env. Hooks: ", hookSource)
	}

	container = getContainerFromFile("test/container.json")
	hookSource = "teest_123_hash_xyz:start,restart:pipework:eth0:{{.ID}}:192.168.242.1/24"
	hooks = getHooks(hookSource)
	events = GetEvents(hooks, &container)

	if (events != nil) {
		t.Error("Container shouldn't be found by name", container.Name, "hooks: ", hookSource)
	}

	hookSource = "LIBVIRT_SERVICE_PORT=16599:start,restart:pipework:eth0:{{.ID}}:192.168.242.1/24"
	hooks = getHooks(hookSource)
	events = GetEvents(hooks, &container)

	if (events != nil) {
		t.Error("Container shouldn't be found by env. Hooks: ", hookSource)
	}
}

func getHooks(source string) hookMap {
	result := hookMap{}
	result.Set(source)
	return result
}

func getContainerFromFile(filename string) Container {

	result := Container{}
	reader, _ := os.Open(filename)
	decoder := json.NewDecoder(reader)
	decoder.Decode(&result)
	return result
}

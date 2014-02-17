package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/dotcloud/docker"
	"github.com/dotcloud/docker/utils"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/exec"
	"strings"
)

const APIVERSION = "1.8"

var (
	proto  = flag.String("proto", "unix", "protocol to use")
	addr   = flag.String("addr", "/var/run/docker.sock", "address to connect to")
	since  = flag.String("since", "1", "watch for events since given value in seconds since epoch")
	replay = flag.String("replay", "", "file to use to simulate/replay events from. Format = docker events")
	debug  = flag.Bool("v", false, "verbose logging")
	hm     hookMap
)

// id, event, command
type hookMap map[string]map[string][]string

func (hm hookMap) String() string { return "" }
func (hm hookMap) Set(str string) error {
	parts := strings.Split(str, ":")
	if len(parts) < 3 {
		return fmt.Errorf("Couldn't parse %s", str)
	}
	id := parts[0]
	event := parts[1]
	command := parts[2:]
	log.Printf("= %s:%s:%s", id, event, command)

	if hm[id] == nil {
		hm[id] = make(map[string][]string)
	}
	hm[id][event] = command
	return nil
}

func getContainer(id string) (*docker.Container, error) {
	resp, err := request("/containers/" + id + "/json")
	if err != nil {
		return nil, fmt.Errorf("Couldn't find container %s: %s", id, err)
	}
	defer resp.Body.Close()
	container := &docker.Container{}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return container, json.Unmarshal(body, &container)
}

func request(path string) (*http.Response, error) {
	req, err := http.NewRequest("GET", path, nil)
	if err != nil {
		return nil, err
	}

	conn, err := net.Dial(*proto, *addr)
	if err != nil {
		return nil, err
	}

	clientconn := httputil.NewClientConn(conn, nil)
	resp, err := clientconn.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 400 {
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}
		if len(body) == 0 {
			return nil, fmt.Errorf("Error: %s", http.StatusText(resp.StatusCode))
		}

		return nil, fmt.Errorf("HTTP %s: %s", http.StatusText(resp.StatusCode), body)
	}
	return resp, nil
}

func main() {
	hm = hookMap{}
	flag.Var(&hm, "e", "specify hook map in format container:event:command[:arg1:arg2...], arg == {{ID}} will be replaced by container ID")
	flag.Parse()
	if len(hm) == 0 {
		fmt.Fprintf(os.Stderr, "Please set hooks via -e flag\n")
		flag.PrintDefaults()
		os.Exit(1)
	}

	v := url.Values{}
	v.Set("since", *since)

	resp, err := request("/events?" + v.Encode())
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()
	if *replay != "" {
		file, err := os.Open(*replay)
		if err != nil {
			log.Fatalf("Couldn't replay from file %s: %s", *replay, err)
		}
		watch(file)
	} else {
		watch(resp.Body)
	}
}

func watch(r io.Reader) {
	dec := json.NewDecoder(r)
	for {
		event := utils.JSONMessage{}
		if err := dec.Decode(&event); err != nil {
			if err == io.EOF {
				break
			}
			log.Fatalf("Couldn't decode message: %s", err)
		}
		if *debug {
			log.Printf("< %s:%s", event.ID, event.Status)
		}
		container, err := getContainer(event.ID)
		if err != nil {
			log.Printf("Warning: Couldn't get container %s: %s", event.ID, err)
			continue
		}
		events := hm[event.ID]
		if events == nil {
			events = hm[strings.TrimLeft(container.Name, "/")]
			if events == nil {
				continue
			}
		}
		c := events[event.Status]
		if len(c) == 0 {
			continue
		}
		args := []string{}
		for _, arg := range c[1:] {
			if arg == "{{ID}}" {
				arg = event.ID
			}
			args = append(args, arg)
		}

		command := exec.Command(c[0], args...)
		log.Printf("> %s [ %v ]", command.Path, command.Args[1:])
		out, err := command.CombinedOutput()
		if err != nil {
			log.Printf("! ERROR %s: %s", err, out)
			continue
		}
		if out != nil {
			log.Printf("- %s", out)
		}
	}
}

package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
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
	"text/template"

	"github.com/docker/docker/utils"
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

type Container struct {
	Name  string
	ID    string
	Event utils.JSONMessage
}

// id, event, command
type hookMap map[string]map[string][][]*template.Template

func (hm hookMap) String() string { return "" }
func (hm hookMap) Set(str string) error {
	parts := strings.Split(str, ":")
	if len(parts) < 3 {
		return fmt.Errorf("Couldn't parse %s", str)
	}
	id := parts[0]
	events := strings.Split(parts[1], ",")
	command, err := parseTemplates(parts[2:])
	if err != nil {
		return err
	}

	if hm[id] == nil {
		hm[id] = make(map[string][][]*template.Template)
	}
	for _, event := range events {
		log.Printf("= %s:%s:%s", id, event, str)
		hm[id][event] = append(hm[id][event], command)
	}
	return nil
}

func parseTemplates(templates []string) ([]*template.Template, error) {
	tl := []*template.Template{}
	for i, t := range templates {
		tmpl, err := template.New(fmt.Sprintf("t-%d", i)).Parse(t)
		if err != nil {
			return nil, err
		}
		tl = append(tl, tmpl)
	}
	return tl, nil
}

func getContainer(event utils.JSONMessage) (*Container, error) {
	resp, err := request("/containers/" + event.ID + "/json")
	if err != nil {
		return nil, fmt.Errorf("Couldn't find container for event %#v: %s", event, err)
	}
	defer resp.Body.Close()
	container := &Container{}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	container.Event = event
	container.ID = event.ID
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
	flag.Var(&hm, "e", "Hook map with template text executed in docker event (see JSONMessage) context, format: container:event[,event]:command[:arg1:arg2...]")
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
		container, err := getContainer(event)
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
		commands := events[event.Status]
		if len(commands) == 0 {
			continue
		}
		for _, command := range commands {
			if len(command) == 0 {
				continue
			}
			args := []string{}
			for _, template := range command {
				buf := bytes.NewBufferString("")
				if err := template.Execute(buf, container); err != nil {
					log.Fatalf("Couldn't render template: %s", err)
				}
				args = append(args, buf.String())
			}

			command := exec.Command(args[0], args[1:]...)
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
}

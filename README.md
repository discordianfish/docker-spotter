# docker-spotter

docker-spotter connects to a docker daemon, receives events and
executes commands provided on the command line.

## Usage

    -addr="/var/run/docker.sock": address to connect to
    -e=: Hook map with template text executed in docker event (see JSONMessage) context,
         format: container:event[,event]:command[:arg1:arg2...]
    -proto="unix": protocol to use
    -replay="": file to use to simulate/replay events from. Format = docker events
    -since="1": watch for events since given value in seconds since epoch
    -v=false: verbose logging

The command and each parameter get parsed as
[text/template](http://golang.org/pkg/text/template/) and will get
rendered with {{.Name}} set to the containers name, {{.ID}} to it's ID
and {{.Event}} to the [JSONMessage](http://godoc.org/github.com/dotcloud/docker/utils#JSONMessage)
which triggered the event.


## Example

This example will run `pipework eth0 <id> 192.168.242.1/24` when a
container named 'pxe-server' starts or restarts and `echo gone` when it stops.

    ./spotter \
      -e 'pxe-server:start,restart:pipework:eth0:{{.ID}}:192.168.242.1/24' \
      -e 'pxe-server:stop:echo:gone'



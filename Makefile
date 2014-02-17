VERSION := $(shell cat VERSION)
TAG     := v$(VERSION)

all: build

docker-spotter:
	go build

build: docker-spotter

tag:
	git tag $(TAG)
	git push --tags

release: build tag
	./release.sh $(TAG) discordianfish/docker-spotter docker-spotter


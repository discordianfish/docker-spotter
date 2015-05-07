FROM       alpine
MAINTAINER Johannes 'fish' Ziemke <fish@docker.com> (@discordianfish)

ENV  GOPATH /go
ENV APPPATH $GOPATH/src/github.com/docker-infra/docker-spotter
COPY . $APPPATH
RUN apk add --update -t build-deps go git libc-dev gcc libgcc \
    && cd $APPPATH && go get -d && go build -o /bin/docker-spotter \
    && mkdir /docker-spotter \
    && ln -s /bin/docker-spotter /docker-spotter/docker-spotter \
    && apk del --purge build-deps && rm -rf $GOPATH

WORKDIR    /docker-spotter
ENTRYPOINT [ "/bin/docker-spotter" ]

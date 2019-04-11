FROM golang:1.12

ADD . /go/src/github.com/IntelAI/nodus
WORKDIR /go/src/github.com/IntelAI/nodus

RUN go get -u github.com/golang/dep/cmd/dep
RUN make install_dependencies
RUN make install

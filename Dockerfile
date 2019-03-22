FROM golang:1.11

ADD . /go/src/github.com/IntelAI/nodus
WORKDIR /go/src/github.com/IntelAI/nodus
RUN curl -LO https://storage.googleapis.com/kubernetes-release/release/$(curl -s https://storage.googleapis.com/kubernetes-release/release/stable.txt)/bin/linux/amd64/kubectl \
    && chmod +x ./kubectl \
    && mv ./kubectl /usr/local/bin/kubectl

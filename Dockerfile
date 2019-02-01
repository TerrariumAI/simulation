FROM golang:1.9.1

WORKDIR /go/src/github.com/olamai/simulation/server
COPY server .

RUN go get -v ./...
RUN go install -v ./...

EXPOSE 7771

CMD [ "server" ]
# stage 1: build
FROM golang:1.10 as builder
LABEL stage=intermediate

COPY . $GOPATH/src/github.com/xelalexv/dregsy/
WORKDIR $GOPATH/src/github.com/xelalexv/dregsy/

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -v -a \
	-tags netgo -installsuffix netgo -ldflags '-w' \
	-o /go/bin/dregsy ./cmd/dregsy/

# stage 2: create actual dregsy container
FROM scratch

COPY --from=builder /go/bin/dregsy /

# also get CA certificates
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

CMD ["/dregsy", "-config=config.yaml"]

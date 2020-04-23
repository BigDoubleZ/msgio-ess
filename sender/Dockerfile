FROM golang

WORKDIR /go/src/msgio-ess/
COPY . .
RUN go mod download
RUN go build -ldflags "-linkmode external -extldflags -static" -a -o bin/sender ./sender/sender.go

FROM scratch
COPY --from=0 /go/src/msgio-ess/bin/sender /sender
CMD ["/sender"]

# msgio-ess $
# sudo docker build -t ess-sender -f sender/Dockerfile .
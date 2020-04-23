FROM golang

WORKDIR /go/src/msgio-ess/
COPY . .
RUN go mod download
RUN go build -ldflags "-linkmode external -extldflags -static" -a -o bin/acceptor ./acceptor/acceptor.go

FROM scratch
COPY --from=0 /go/src/msgio-ess/bin/acceptor /acceptor
EXPOSE 8080
CMD ["/acceptor"]

# : msgio-ess$ sudo docker build -t ess-acceptor -f acceptor/Dockerfile .
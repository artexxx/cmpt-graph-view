FROM golang:1.22-alpine3.20

RUN apk add --no-cache git ca-certificates

WORKDIR /src
COPY . .
RUN go build -trimpath -ldflags="-s -w" -o /bin/linkmap-gen ./

ENTRYPOINT ["/bin/linkmap-gen"]

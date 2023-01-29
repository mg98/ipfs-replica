FROM golang:1.19-alpine
RUN apk add build-base
WORKDIR /app
COPY go.* ./
RUN go mod download
COPY *.go ./
RUN go build .
CMD ["/app/ipfs-replicate"]
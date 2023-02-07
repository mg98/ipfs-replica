FROM golang:1.19-alpine
RUN apk add build-base
WORKDIR /app
COPY . .
RUN go mod download
RUN go mod verify
RUN go build -o ipfs_replicate
RUN chmod +x ipfs_replicate
CMD ["/app/ipfs_replicate"]

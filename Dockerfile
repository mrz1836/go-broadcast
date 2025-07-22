# Example Dockerfile for a Go application (this is just a placeholder)
FROM scratch
COPY go-broadcast /
ENTRYPOINT ["/go-broadcast"]

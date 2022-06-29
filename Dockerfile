FROM golang:1.18 as deps

ENV CGO_ENABLED=0 \
    GOOS=linux \
    GOARCH=amd64

WORKDIR /build

# Copy and download dependencies
COPY go.mod .
COPY go.sum .
RUN go mod download

# Copy the code into the container
COPY query/ ./query/
COPY scripts/ ./scripts/

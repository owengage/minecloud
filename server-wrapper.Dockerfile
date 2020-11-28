FROM golang:1.14-alpine3.11 as builder

WORKDIR /app

COPY go.mod .
COPY go.sum .
RUN go mod download

COPY cmd cmd/
COPY pkg pkg/
RUN go build -o serverwrapper cmd/serverwrapper/*.go

# Build final image.
FROM openjdk:8-alpine

COPY --from=builder /app/serverwrapper .
ENTRYPOINT [ "./serverwrapper", "-jar", "/server/fabric-server-launch.jar", "-address", "0.0.0.0:80" ]

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
RUN wget https://launcher.mojang.com/v1/objects/bb2b6b1aefcd70dfd1892149ac3a215f6c636b07/server.jar
COPY --from=builder /app/serverwrapper .
ENTRYPOINT [ "./serverwrapper", "-jar", "server.jar", "-address", "0.0.0.0:80" ]

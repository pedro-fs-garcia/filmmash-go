FROM golang:1.26.4-trixie AS build
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /app ./cmd/filmmash-go

FROM golang:1.26.4-alpine
COPY --from=build /app /app
VOLUME ["/logs"]
EXPOSE 8000
ENTRYPOINT ["/app"]

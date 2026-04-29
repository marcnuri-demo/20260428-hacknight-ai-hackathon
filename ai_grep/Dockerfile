FROM golang:1.23-alpine AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY main.go ./
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/ai_grep .

FROM scratch
COPY --from=builder /out/ai_grep /ai_grep
ENTRYPOINT ["/ai_grep"]

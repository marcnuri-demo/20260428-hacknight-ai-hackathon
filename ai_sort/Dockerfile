FROM golang:1.26-alpine AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY main.go ./
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/ai_sort .

FROM scratch
COPY --from=builder /out/ai_sort /ai_sort
ENTRYPOINT ["/ai_sort"]

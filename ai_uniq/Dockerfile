FROM golang:1.23-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY main.go ./
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /out/ai_uniq .

FROM scratch
COPY --from=build /out/ai_uniq /ai_uniq
ENTRYPOINT ["/ai_uniq"]

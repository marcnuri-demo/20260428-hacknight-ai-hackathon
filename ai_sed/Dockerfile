FROM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY *.go ./
RUN CGO_ENABLED=0 go build -trimpath -ldflags='-s -w' -o /out/ai_sed .

FROM scratch
COPY --from=build /out/ai_sed /ai_sed
ENTRYPOINT ["/ai_sed"]

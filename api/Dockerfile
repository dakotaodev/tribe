FROM golang:1.25-alpine AS build

WORKDIR /src
COPY go.mod ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/tribe-api ./cmd/server

FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=build /out/tribe-api /tribe-api
EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/tribe-api"]

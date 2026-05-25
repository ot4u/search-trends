FROM golang:1.25.1 AS build

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o /out/search-trends ./cmd/app

FROM gcr.io/distroless/static-debian12:nonroot

WORKDIR /app

COPY --from=build /out/search-trends /app/search-trends

EXPOSE 8080

ENTRYPOINT ["/app/search-trends"]

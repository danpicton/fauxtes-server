FROM golang:1.24-bookworm AS build

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=1 go build -ldflags='-s -w -extldflags "-static"' -o /fauxtes ./cmd/fauxtes

FROM gcr.io/distroless/static-debian12

COPY --from=build /fauxtes /fauxtes

ENV FAUXTES_ADDR=:8080
ENV FAUXTES_DB=/data/fauxtes.db

EXPOSE 8080

VOLUME /data

ENTRYPOINT ["/fauxtes"]

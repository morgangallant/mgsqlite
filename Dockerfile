# Build step.
FROM golang:1.16 as build
ADD . /mgsqlite
WORKDIR /mgsqlite/
RUN go build -o mgsqlite .
ENTRYPOINT ["/mgsqlite/mgsqlite"]
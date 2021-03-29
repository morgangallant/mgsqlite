# Build step.
FROM golang:1.16 as build
ADD . /mgsqlite
WORKDIR /mgsqlite/
RUN go build -o mgsqlite .

# Run step (don't think zeet.co supports distroless or scratch).
FROM alpine
WORKDIR /mg
COPY --from=build /mgsqlite/mgsqlite /mg/
ENTRYPOINT [ "/mg/mgsqlite" ]
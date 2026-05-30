FROM golang:1.23-alpine AS builder
WORKDIR /src
COPY cmd/oracle/go.mod cmd/oracle/go.sum ./
RUN go mod download
COPY cmd/oracle/ ./
RUN CGO_ENABLED=0 go build -o /arcana-oracle .

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=builder /arcana-oracle /arcana-oracle
EXPOSE 8089
ENTRYPOINT ["/arcana-oracle"]

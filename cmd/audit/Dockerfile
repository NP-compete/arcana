FROM golang:1.23-alpine AS builder
WORKDIR /src
COPY cmd/audit/go.mod ./
RUN go mod download
COPY cmd/audit/ ./
RUN CGO_ENABLED=0 go build -o /arcana-audit .

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=builder /arcana-audit /arcana-audit
EXPOSE 8100
ENTRYPOINT ["/arcana-audit"]

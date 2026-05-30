FROM golang:1.23-alpine AS builder
WORKDIR /src
COPY cmd/operator/go.mod ./
RUN go mod download
COPY cmd/operator/ ./
RUN CGO_ENABLED=0 go build -o /arcana-operator .

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=builder /arcana-operator /arcana-operator
EXPOSE 8082
ENTRYPOINT ["/arcana-operator"]

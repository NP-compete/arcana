FROM golang:1.23-alpine AS builder
WORKDIR /src
COPY cmd/scheduler/go.mod ./
RUN go mod download
COPY cmd/scheduler/ ./
RUN CGO_ENABLED=0 go build -o /arcana-scheduler .

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=builder /arcana-scheduler /arcana-scheduler
EXPOSE 8103
ENTRYPOINT ["/arcana-scheduler"]

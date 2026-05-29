FROM golang:1.23-alpine AS builder
WORKDIR /build
COPY go.work go.work
COPY cmd/agui/ cmd/agui/
COPY pkg/ pkg/
RUN cd cmd/agui && go build -o /arcana-agui .

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=builder /arcana-agui /arcana-agui
EXPOSE 8084
ENTRYPOINT ["/arcana-agui"]

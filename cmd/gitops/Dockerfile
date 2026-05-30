FROM golang:1.23-alpine AS builder
WORKDIR /src
COPY cmd/gitops/go.mod cmd/gitops/go.sum ./
RUN go mod download
COPY cmd/gitops/ ./
RUN CGO_ENABLED=0 go build -o /arcana-gitops .

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=builder /arcana-gitops /arcana-gitops
EXPOSE 8106
ENTRYPOINT ["/arcana-gitops"]

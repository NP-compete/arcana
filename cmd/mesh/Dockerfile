FROM golang:1.23-alpine AS builder
WORKDIR /build
COPY go.work go.work
COPY cmd/mesh/ cmd/mesh/
COPY pkg/ pkg/
RUN cd cmd/mesh && go build -o /arcana-mesh .

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=builder /arcana-mesh /arcana-mesh
EXPOSE 8083
ENTRYPOINT ["/arcana-mesh"]

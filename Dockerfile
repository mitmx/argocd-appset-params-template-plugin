FROM golang:1.22-alpine AS build
WORKDIR /src
COPY go.mod ./
RUN go mod download
COPY . .
ARG TARGETOS=linux
ARG TARGETARCH=amd64
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -trimpath -ldflags="-s -w" -o /out/argocd-appset-params-template-plugin ./cmd/server

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/argocd-appset-params-template-plugin /argocd-appset-params-template-plugin
EXPOSE 4355
ENTRYPOINT ["/argocd-appset-params-template-plugin"]

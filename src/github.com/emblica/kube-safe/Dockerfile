FROM golang:1.11 AS builder

RUN curl -fsSL -o /usr/local/bin/dep https://github.com/golang/dep/releases/download/v0.5.0/dep-linux-amd64 && chmod +x /usr/local/bin/dep

RUN mkdir -p /go/src/github.com/emblica/kube-safe
WORKDIR /go/src/github.com/emblica/kube-safe

COPY Gopkg.toml Gopkg.lock ./
# copies the Gopkg.toml and Gopkg.lock to WORKDIR

RUN dep ensure -vendor-only
# install the dependencies without checking for go code
COPY *.go ./

RUN CGO_ENABLED=0 go build \
    -installsuffix 'static' \
    -o kube-safe-maxpods .


FROM scratch AS final


# Import the Certificate-Authority certificates for enabling HTTPS.
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Import the compiled executable from the first stage.
COPY --from=builder /go/src/github.com/emblica/kube-safe/kube-safe-maxpods /kube-safe-maxpods


# Run the compiled binary.
ENTRYPOINT ["/kube-safe-maxpods"]

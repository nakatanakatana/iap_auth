FROM golang:1.16 AS builder

WORKDIR /app/source
COPY ./ /app/source

ARG CGO_ENABLED=0
ARG GOOS=linux
ARG GOARCH=amd64

RUN go build -o ./out/iap_auth .


FROM busybox
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /app/source/out/iap_auth .

ENV PORT=8081 \
	LOGGER_LEVEL=INFO \
  REFRESH_TIME_SECONDS= \
	IAP_HOST= \
	SERVICE_ACCOUNT_CREDENTIALS= \
	CLIENT_ID=
EXPOSE ${PORT}
CMD ./iap_auth

FROM golang:1.19-alpine AS build_base

RUN apk add --no-cache git

WORKDIR /app

COPY . .

RUN go mod download

#RUN CGO_ENABLED=0 go test -v

RUN go build -o ./out/slackqueue ./cmd/slackqueue

FROM alpine:3.9 
RUN apk add ca-certificates

COPY --from=build_base /app/out/slackqueue /app/slackqueue

ENV SLACK_TOKEN=
ENV SLACK_SIGNING_SECRET=
ENV GOOGLE_PROJECT_ID=
ENV PORT=8080

EXPOSE $PORT

CMD ["/app/slackqueue"]

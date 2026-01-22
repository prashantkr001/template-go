FROM golang:1.24
LABEL owner=""

RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates  \
    git \
    libssl-dev \
    netbase \
    && apt-get autoremove -y && apt-get autoclean -y

RUN go install github.com/air-verse/air@latest

WORKDIR /app

# Below line's required if you're using private Go repositories. It can either be ignored or removed
# if not using any private repositories.
ENV GOPRIVATE="github.com/prashantkr001"
RUN git config --global url."https://${GITHUB_TOKEN}:x-oauth-basic@github.com/".insteadOf "https://github.com/"

ENV ENVIRONMENT="development"

CMD ["air", "-c", "/etc/.air.toml"]
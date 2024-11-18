FROM debian:bookworm-slim

ENV DEBIAN_FRONTEND noninteractive

RUN apt-get update -y && \
    apt-get -y install file wget ca-certificates curl jq coreutils gnupg --no-install-recommends \
    && rm -rf /var/lib/apt/lists/*

# Install Gcloud CLI (https://cloud.google.com/sdk/docs/install)
RUN echo "deb [signed-by=/usr/share/keyrings/cloud.google.gpg] https://packages.cloud.google.com/apt cloud-sdk main" \
    | tee -a /etc/apt/sources.list.d/google-cloud-sdk.list \
    && curl https://packages.cloud.google.com/apt/doc/apt-key.gpg | gpg --dearmor -o /usr/share/keyrings/cloud.google.gpg \
    && apt-get update -y && apt-get install google-cloud-cli google-cloud-sdk-package-go-module -y --no-install-recommends \
    && rm -rf /var/lib/apt/lists/*

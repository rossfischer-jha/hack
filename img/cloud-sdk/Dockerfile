FROM gcr.io/google.com/cloudsdktool/cloud-sdk:slim

ARG TARGETARCH
ARG YQ_VERSION=v4.44.3

RUN curl --fail --silent --show-error --location \
      "https://github.com/mikefarah/yq/releases/download/${YQ_VERSION}/yq_linux_${TARGETARCH}" \
      --output /usr/local/bin/yq \
    && chmod +x /usr/local/bin/yq

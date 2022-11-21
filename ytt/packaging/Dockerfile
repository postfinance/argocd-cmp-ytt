FROM curlimages/curl as ytt-download

ARG TARGETOS
ARG TARGETARCH

RUN curl --location --insecure https://github.com/vmware-tanzu/carvel-ytt/releases/download/v0.44.0/ytt-${TARGETOS}-${TARGETARCH} > /tmp/ytt

FROM argoproj/argocd:v2.5.2
USER root

COPY --from=ytt-download /tmp/ytt /usr/local/bin/ytt
RUN chmod +x /usr/local/bin/ytt

COPY packaging/plugin.yaml /home/argocd/cmp-server/config/plugin.yaml
COPY cmp-ytt /home/argocd/cmp-ytt

USER 999
ENTRYPOINT [ "/var/run/argocd/argocd-cmp-server" ]

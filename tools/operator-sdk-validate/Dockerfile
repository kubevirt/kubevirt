FROM quay.io/fedora/fedora:33-x86_64

COPY deploy/olm-catalog/community-kubevirt-hyperconverged /manifests
COPY tools/operator-sdk-validate/validate-bundles.sh .

RUN ./validate-bundles.sh

ENTRYPOINT ["operator-sdk"]
CMD ["--help"]

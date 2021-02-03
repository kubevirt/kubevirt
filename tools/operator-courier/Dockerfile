FROM registry.access.redhat.com/ubi8/python-36

RUN pip3 install operator-courier
COPY deploy/olm-catalog/community-kubevirt-hyperconverged /manifests

RUN operator-courier verify --ui_validate_io /manifests

ENTRYPOINT ["operator-courier"]
CMD ["--help"]

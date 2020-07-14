FROM registry.access.redhat.com/ubi8/python-36

RUN pip3 install --upgrade operator-courier==2.1.7
COPY deploy/olm-catalog/kubevirt-hyperconverged /manifests

RUN operator-courier verify --ui_validate_io /manifests

ENTRYPOINT ["operator-courier"]
CMD ["--help"]

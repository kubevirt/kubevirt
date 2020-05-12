FROM centos:centos8.1.1911

COPY deploy/nightly-bundle/deploy.sh /deploy.sh
COPY deploy/nightly-bundle/create_docker_config.sh /create_docker_config.sh
COPY deploy/nightly-bundle/kubevirt-testing-infra.yaml /kubevirt-testing-infra.yaml
COPY deploy/nightly-bundle/kubevirt-tests-pod-spec-override.json.in /kubevirt-tests-pod-spec-override.json.in

RUN yum install -y jq sed
RUN chmod +x /deploy.sh /create_docker_config.sh

FROM gcr.io/k8s-testimages/bootstrap:v20190516-c6832d9
RUN apt update && apt install -y kmod pciutils
RUN wget https://dl.google.com/go/go1.12.6.linux-amd64.tar.gz && tar -C /usr/local -xzf go1.12.6.linux-amd64.tar.gz
ENV PATH="/usr/local/go/bin:${PATH}"
RUN GO111MODULE="on" go get sigs.k8s.io/kind@v0.2.1



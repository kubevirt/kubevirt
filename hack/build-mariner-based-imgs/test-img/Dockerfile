FROM mcr.microsoft.com/cbl-mariner/base/core:2.0

ARG A="A"

#FROM mcr.microsoft.com/cbl-mariner/base/core:2.0@sha256:997710e0e84781c1243f927ceafe0f1082d09e318b4f9f222bd82d057b4470cf
WORKDIR /workspace

ARG TARGETARCH="amd64"

RUN tdnf -y install mariner-repos-extended.noarch
RUN tdnf -y install mariner-repos-preview
RUN tdnf -y install \
  augeas 

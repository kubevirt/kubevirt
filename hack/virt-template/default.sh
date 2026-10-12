#!/bin/bash

# virt_template_*_digest values are the multi-arch manifest list digests.
virt_template_version=${VIRT_TEMPLATE_VERSION:-"v0.3.0"}
virt_template_yaml_sha256=${VIRT_TEMPLATE_YAML_SHA256:-"f09c9293ed81986c6cc3991ba3991f5f21e55a4b5e7183d21352794b8bf25779"}
virt_template_apiserver_digest=${VIRT_TEMPLATE_APISERVER_DIGEST:-"sha256:eaf118b2c1645c197f351c4aa0b56e49ace5c438530803b3ebcf390dc7b43731"}
virt_template_controller_digest=${VIRT_TEMPLATE_CONTROLLER_DIGEST:-"sha256:5313fe9a7364881a8afd9ac3afb764fa7d009e098f4180417ed3708c5bdf2efc"}

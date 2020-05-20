#!/usr/bin/env bash
set -ex

TMP_PLANTUML_DIR=$(mktemp -d)
wget -O "${TMP_PLANTUML_DIR}/plantuml.jar" http://sourceforge.net/projects/plantuml/files/plantuml.jar/download
java -jar "${TMP_PLANTUML_DIR}/plantuml.jar" -checkmetadata -tsvg pkg/controller/hyperconverged/hyperconverged_controller.go
rm -rf "${TMP_PLANTUML_DIR}"

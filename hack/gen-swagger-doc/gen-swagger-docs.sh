#!/usr/bin/env bash

# gen-swagger-docs.sh $API_VERSION $OUTPUT_FORMAT
# API_VERSION=v1
# OUTPUT_FORMAT=html|markdown

source $(dirname "$0")/../../hack/common.sh

set -o errexit
set -o nounset
set -o pipefail
set -x

VERSION="$1"
OUTPUT_FORMAT="$2"

VERSION="${VERSION:-v1}"
OUTPUT_FORMAT="${OUTPUT_FORMAT:-html}"
GIT_REPO_LINK="https://github.com/kubevirt/kubevirt"
if [ "$OUTPUT_FORMAT" = "html" ]; then
    SUFFIX="adoc"
    HEADER="="
    LINK1_TEMPLATE="\* \<\<\${VERSION}.\$m\>\>"
    LINK_DEFINITIONS="* link:./definitions.html[Types Definition]"
    LINK_OPERATIONS="* link:./operations.html[Operations]"
    GRADLE_EXTRA_PARAMS=""
elif [ "$OUTPUT_FORMAT" = "markdown" ]; then
    SUFFIX="md"
    HEADER="#"
    LINK1_TEMPLATE="\* [\${VERSION}.\$m]\(definitions.md#\${VERSION}-\${m,,}\)"
    LINK_DEFINITIONS="* [Types Definition](definitions.md)"
    LINK_OPERATIONS="* [Operations](operations.md)"
    GRADLE_EXTRA_PARAMS="-PmarkupLanguage=MARKDOWN"
else
    echo "Unknown OUTPUT_FORMAT=${OUTPUT_FORMAT}"
    exit 1
fi
WORKDIR="hack/gen-swagger-doc"
GRADLE_BUILD_FILE="$WORKDIR/build.gradle"

# Generate *.adoc files from swagger.json
gradle -b $GRADLE_BUILD_FILE $GRADLE_EXTRA_PARAMS convertSwagger2markup --info

#insert a TOC for top level API objects
buf="${HEADER}${HEADER} Top Level API Objects\n\n"
top_level_models=$(grep '&[A-Za-z]*{},' staging/src/kubevirt.io/api/core/${VERSION}/types.go | sed 's/.*&//;s/{},//')

# check if the top level models exist in the definitions.$SUFFIX. If they exist,
# their name will be <version>.<model_name>
for m in $top_level_models; do
    if grep -xq "${HEADER}${HEADER}${HEADER} ${VERSION}.$m" "$WORKDIR/definitions.${SUFFIX}"; then
        buf+="$(eval echo $LINK1_TEMPLATE)\n"
    fi
done
sed -i "1i $buf" "$WORKDIR/definitions.${SUFFIX}"

# change the title of paths.adoc from "paths" to "operations"
sed -i "s|${HEADER}${HEADER} Paths|${HEADER}${HEADER} Operations|g" "$WORKDIR/paths.${SUFFIX}"
mv -f "$WORKDIR/paths.${SUFFIX}" "$WORKDIR/operations.${SUFFIX}"

# Add links to definitons & operations under overview
cat >>"$WORKDIR/overview.${SUFFIX}" <<__END__
${HEADER}${HEADER} KubeVirt API Reference

${LINK_DEFINITIONS}
${LINK_OPERATIONS}
__END__

if [ "$OUTPUT_FORMAT" = "html" ]; then
    # $$ has special meaning in asciidoc, we need to escape it
    sed -i 's|\$\$|+++$$+++|g' "$WORKDIR/definitions.adoc"
    sed -i 's|```||g' "$WORKDIR/definitions.adoc"
    sed -i '1 i\:last-update-label!:' "$WORKDIR/"*.adoc

    # Determine version of KubeVirt, as a commit hash or tag in case of tagged commit.
    gittagmatch="$(git describe --exact-match 2>/dev/null || true)"
    if [ -n "$gittagmatch" ]; then
        gitcommithash="${gittagmatch}"
        gitlink="${GIT_REPO_LINK}/releases/tag"
    else
        gitcommithash="$(git rev-parse HEAD || echo no-revision)"
        gitlink="${GIT_REPO_LINK}/commit"
    fi
    sed -i -e "/KubeVirt API\$/a\\:revnumber: ${gitcommithash}" \
        -e "/__Terms of service__ :/a\\__Version__ : ${gitlink}/{revnumber}[{revnumber}]" \
        "$WORKDIR/overview.adoc"

    # Generate *.html files from *.adoc
    rm -rf "$WORKDIR/html5" && mkdir -p "$WORKDIR/html5"
    adoc_files=("definitions.adoc" "overview.adoc" "security.adoc" "operations.adoc")
    for html_file in ${adoc_files[@]}; do
        asciidoctor \
            --failure-level INFO \
            --attribute toc=right \
            --destination-dir $WORKDIR/html5 \
            $PWD/$WORKDIR/$html_file
    done

    rm -rf "$WORKDIR/html5/content" && mkdir "$WORKDIR/html5/content" && mv -f "$WORKDIR/html5/"*.html "$WORKDIR/html5/content"
    mv -f "$WORKDIR/html5/content/overview.html" "$WORKDIR/html5/content/index.html"
elif [ "$OUTPUT_FORMAT" = "markdown" ]; then
    # Generate TOC for definitions & operations as README.md
    cd "$WORKDIR"
    echo "# KubeVirt API Reference" >README.md
    # reference to master is for an external repo and can't yet be changed
    curl \
        https://raw.githubusercontent.com/ekalinin/github-markdown-toc/master/gh-md-toc |
        bash -s "definitions.md" "operations.md" |
        sed 's/^      //' >>"README.md"
    cd -
fi

mkdir -p ${APIDOCS_OUT_DIR}/html
mv ${WORKDIR}/html5/content/* ${APIDOCS_OUT_DIR}/html
mv ${WORKDIR}/*.adoc ${APIDOCS_OUT_DIR}/
echo "SUCCESS"

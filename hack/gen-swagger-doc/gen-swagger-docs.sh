#!/bin/bash

set -o errexit
set -o nounset
set -o pipefail

VERSION="${1:-v1}"
WORKDIR="hack/gen-swagger-doc"
GRADLE_BUILD_FILE="$WORKDIR/build.gradle"

# Generate *.adoc files from swagger.json
gradle -b $GRADLE_BUILD_FILE gendocs --info

#insert a TOC for top level API objects
buf="== Top Level API Objects\n\n"
top_level_models=$(grep '&[A-Za-z]*{},' pkg/api/${VERSION}/types.go | sed 's/.*&//;s/{},//')

# check if the top level models exist in the definitions.adoc. If they exist,
# their name will be <version>.<model_name>
for m in $top_level_models
do
  if grep -xq "=== ${VERSION}.$m" "$WORKDIR/definitions.adoc"
  then
    buf+="* <<${VERSION}.$m>>\n"
  fi
done
sed -i "1i $buf" "$WORKDIR/definitions.adoc"

# fix the links in .adoc, replace <<x,y>> with link:definitions.html#_x[y], and lowercase the _x part
sed -i -e 's|<<\(.*\),\(.*\)>>|link:#\L\1\E[\2]|g' "$WORKDIR/definitions.adoc"
sed -i -e 's|<<\(.*\),\(.*\)>>|link:./definitions.html#\L\1\E[\2]|g' "$WORKDIR/paths.adoc"

# change the title of paths.adoc from "paths" to "operations"
sed -i 's|== Paths|== Operations|g' "$WORKDIR/paths.adoc"
mv -f "$WORKDIR/paths.adoc" "$WORKDIR/operations.adoc"

# $$ has special meaning in asciidoc, we need to escape it
sed -i 's|\$\$|+++$$+++|g' "$WORKDIR/definitions.adoc"

# Add links to definitons & operations under overview
cat >> "$WORKDIR/overview.adoc" << __END__
== KubeVirt API Reference
[%hardbreaks]
* link:./definitions.html[Types Definition]
* link:./operations.html[Operations]

__END__


# Generate *.html files from *.adoc
gradle -b $GRADLE_BUILD_FILE asciidoctor --info

echo "SUCCESS"

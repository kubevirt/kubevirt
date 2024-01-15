#!/usr/bin/bash -ex

base="$1"
increment_type="$2"
RE='[^0-9]*\([0-9]*\)[.]\([0-9]*\)[.]\([0-9]*\)\([0-9A-Za-z-]*\)'

if [[ -z "$increment_type" ]]; then
  increment_type="patch"
fi

MAJOR=$(echo $base | sed -e "s#$RE#\1#")
MINOR=$(echo $base | sed -e "s#$RE#\2#")
PATCH=$(echo $base | sed -e "s#$RE#\3#")

case "$increment_type" in
major)
  ((MAJOR += 1))
  ((MINOR = 0))
  ((PATCH = 0))
  ;;
minor)
  ((MINOR += 1))
  ((PATCH = 0))
  ;;
patch)
  ((PATCH += 1))
  ;;
esac

NEXT_VERSION="$MAJOR.$MINOR.$PATCH"
echo "$NEXT_VERSION"

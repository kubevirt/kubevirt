#!/usr/bin/env bash

OFFENSIVE_WORDS="black[ -]?list|white[ -]?list|master|slave"
ALLOW_LIST=".+/master[a-zA-Z]*/?"

if git grep -inE "${OFFENSIVE_WORDS}" -- ':!vendor' ':!deploy' ':!cluster' ":!${BASH_SOURCE[0]}" | grep -viE "${ALLOW_LIST}"; then
  echo "Validation failed. Found offencive language"
  exit 1
fi

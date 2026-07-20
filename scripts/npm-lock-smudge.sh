#!/usr/bin/env bash

# Keep the public registry URLs when npm is unavailable or already uses npmjs.
NPM_REGISTRY="$(npm config get registry 2>/dev/null || true)"
case "$NPM_REGISTRY" in
    ""|undefined|http://registry.npmjs.org|http://registry.npmjs.org/|https://registry.npmjs.org|https://registry.npmjs.org/)
        cat
        ;;
    *)
        NPM_REGISTRY="${NPM_REGISTRY%/}/"
        sed -E "s#https://registry\.npmjs\.org/#$NPM_REGISTRY#g"
        ;;
esac

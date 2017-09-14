apiserver_tls_crt="$(cat ./cluster/.apiserver.ca.crt 2>/dev/null)" || true

export apiserver_tls_crt

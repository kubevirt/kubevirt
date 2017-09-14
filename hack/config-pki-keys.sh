apiserver_tls_crt="$(cat ./cluster/.apiserver.ca.crt 2>/dev/null)"

export apiserver_tls_crt

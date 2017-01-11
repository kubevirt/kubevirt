source hack/config.sh

pushd tests
go test -master=http://$master_ip:$master_port
popd

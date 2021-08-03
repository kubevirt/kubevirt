#!/bin/bash

CMD=${CMD:-./cluster/kubectl.sh}

function RunCmd {
    cmd=$@
    echo "Command: $cmd"
    echo ""
    bash -c "$cmd"
    stat=$?
    if [ "$stat" != "0" ]; then
        echo "Command failed: $cmd Status: $stat"
    fi
}

function ShowOperatorSummary {

    local kind=$1
    local name=$2
    local namespace=$3

    echo ""
    echo "Status of Operator object: kind=$kind name=$name"
    echo ""

    QUERY="{range .status.conditions[*]}{.type}{'\t'}{.status}{'\t'}{.message}{'\n'}{end}" 
    if [ "$namespace" == "." ]; then
        RunCmd "$CMD get $kind $name -o=jsonpath=\"$QUERY\""
    else
        RunCmd "$CMD get $kind $name -n $namespace -o=jsonpath=\"$QUERY\""
    fi
}

cat <<EOF
=================================
     Start of HCO state dump         
=================================
EOF

if [ -n "${ARTIFACT_DIR}" ]; then
    cat <<EOF
==============================
executing kubevirt-must-gather
==============================

EOF
    mkdir -p ${ARTIFACT_DIR}/kubevirt-must-gather
    RunCmd "${CMD} adm must-gather --image=quay.io/kubevirt/must-gather:latest --dest-dir=${ARTIFACT_DIR}/kubevirt-must-gather --timeout='20m'"
fi

cat <<EOF
==========================
summary of operator status
==========================

EOF

RunCmd "${CMD} get pods -n kubevirt-hyperconverged"
RunCmd "${CMD} get subscription -n kubevirt-hyperconverged -o yaml"
RunCmd "${CMD} get deployment/hco-operator -n kubevirt-hyperconverged -o yaml"
RunCmd "${CMD} get hyperconvergeds -n kubevirt-hyperconverged kubevirt-hyperconverged -o yaml"

ShowOperatorSummary  hyperconvergeds.hco.kubevirt.io kubevirt-hyperconverged kubevirt-hyperconverged

RELATED_OBJECTS=`${CMD} get hyperconvergeds.hco.kubevirt.io kubevirt-hyperconverged -n kubevirt-hyperconverged -o go-template='{{range .status.relatedObjects }}{{if .namespace }}{{ printf "%s %s %s\n" .kind .name .namespace }}{{ else }}{{ printf "%s %s .\n" .kind .name }}{{ end }}{{ end }}'`

echo "${RELATED_OBJECTS}" | while read line; do 

    fields=( $line )
    kind=${fields[0]} 
    name=${fields[1]} 
    namespace=${fields[2]} 

    if [ "$kind" != "ConfigMap" ]; then
        ShowOperatorSummary $kind $name $namespace
    fi
done

cat <<EOF

======================
ClusterServiceVersions
======================
EOF

RunCmd "${CMD} get clusterserviceversions -n kubevirt-hyperconverged"
RunCmd "${CMD} get clusterserviceversions -n kubevirt-hyperconverged -o yaml"

cat <<EOF

============
InstallPlans
============
EOF

RunCmd "${CMD} get installplans -n kubevirt-hyperconverged -o yaml"

cat <<EOF

==============
OperatorGroups
==============
EOF

RunCmd "${CMD} get operatorgroups -n kubevirt-hyperconverged -o yaml"

cat <<EOF

========================
HCO operator related CRD
========================
EOF

echo "${RELATED_OBJECTS}" | while read line; do 

    fields=( $line )
    kind=${fields[0]} 
    name=${fields[1]} 
    namespace=${fields[2]} 

    if [ "$namespace" == "." ]; then
        echo "Related object: kind=$kind name=$name"
        RunCmd "$CMD get $kind $name -o json"
    else
        echo "Related object: kind=$kind name=$name namespace=$namespace"
        RunCmd "$CMD get $kind $name -n $namespace -o json"
    fi
done

cat <<EOF

========
HCO Pods
========

EOF

RunCmd "$CMD get pods -n kubevirt-hyperconverged -o json"

cat <<EOF

=================================
HyperConverged Operator pods logs
=================================
EOF

namespace=kubevirt-hyperconverged
RunCmd "$CMD logs -n $namespace -l name=hyperconverged-cluster-operator"

cat <<EOF

=================================
HyperConverged Webhook pods logs
=================================
EOF
RunCmd "$CMD logs -n $namespace -l name=hyperconverged-cluster-webhook"

cat <<EOF

============
Catalog logs
============
EOF

catalog_namespace=openshift-operator-lifecycle-manager
RunCmd "$CMD logs -n $catalog_namespace $($CMD get pods -n $catalog_namespace | grep catalog-operator | head -1 | awk '{ print $1 }')"


cat <<EOF

===============
HCO Deployments
===============

EOF

RunCmd "$CMD get deployments -n kubevirt-hyperconverged -o json"

cat <<EOF
===============================
     End of HCO state dump    
===============================
EOF

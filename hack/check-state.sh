#!/usr/bin/env bash

echo "CMD: ${CMD}"

if [ "$DEBUG" == "2" ]; then
    set -x
    PS4='${LINENO}: '
fi

EXIT_STATUS=0
MAX_GET_HCO_RETRY=500

HCO_KIND="hyperconvergeds.hco.kubevirt.io"
HCO_NAME="kubevirt-hyperconverged"
HCO_NAMESPACE="kubevirt-hyperconverged"

function get_operator_json() {
    local kind=$1
    local name=$2
    local namespace=$3
    local retry_count=$4
    local retry_count_hco=1
 
    if [ "$namespace" != "." ]; then
        namespace="-n $namespace"
    else
        namespace=""
    fi
    
    while true; do

        if [ "$DEBUG" == "1" ]; then
            echo "${CMD} get $kind $name $namespace"
        fi

        # get values of all possible condition values. Later need to parse it.
        HCO_DATA=`${CMD} get $kind $name $namespace -o go-template='{{range .status.conditions }}{{ if eq .type "ReconcileComplete" }}{{ printf " ReconcileComplete: \"%s\" \"%s\" \"%s\" "  .status .reason .message  }}{{else if eq .type "Progressing" }}{{ printf " ApplicationProgressing: \"%s\" \"%s\" \"%s\" "  .status .reason .message  }}{{ else if eq .type "Available" }}{{ printf "ApplicationAvailable: \"%s\" \"%s\" \"%s\""  .status .reason .message  }}{{ else if eq .type "Degraded" }}{{ printf "ApplicationDegraded: \"%s\" \"%s\" \"%s\""  .status .reason .message  }}{{ end }}{{ end }}'`

        if [ $? -eq 0 ]; then
            break
        fi

        # retry if we failed to get the condition values.
        if [ "$retry_count_hco" -ge "$MAX_GET_HCO_RETRY" ]; then
            echo "can't get operator state kind: $kind name: $name. commmand failed repeatedly. $CMD status: $? "
            exit 1
        fi
        sleep 5
        ((retry_count_hco=retry_count_hco+1))
    done

    HCO_DATA=`printf '%s' "$HCO_DATA" | tr '\n' ' '`
    if [ "$DEBUG" != "" ]; then
       echo "HCO_DATA: $HCO_DATA"
    fi

    # upon first call: check which of the status values occur in the condition values.
    if [ "$retry_count" == "0" ]; then
         has_app_available=`echo "$HCO_DATA" | grep ApplicationAvailable: | wc -l`
         has_app_degraded=`echo "$HCO_DATA" | grep ApplicationDegraded:  | wc -l`
         has_app_progressing=`echo "$HCO_DATA" | grep ApplicationProgressing:  | wc -l`
         has_reconcile=`echo "$HCO_DATA" | grep ReconcileComplete: | wc -l`

         # check that required values are present.
         if [ "$has_app_available" == "0" ] || [ "$has_app_degraded" == "0" ] || [ "$has_app_progressing" == 0 ]; then
             echo "Operator $kind doesn't support required status conditions. Skipping check."
             SKIP_CHECK=1
             return
         fi
    fi
 
    if [ "$has_reconcile" != "0" ]; then
        # if present: extract APPLICATION_AVAILABLE value : check that it has a valid value (either True or False)
        RECONCILE_COMPLETED=`printf '%s' "$HCO_DATA" | sed -e 's/.*ReconcileComplete: "\([^"]*\)".*$/\1/'`
        RECONCILE_COMPLETED_DATA=`printf '%s'  "$HCO_DATA" | sed -e 's/.*ReconcileComplete: "\([^"]*\)" "\([^"]*\)" "\([^"]*\)".*$/Status: \1 Reason: \2 Message: \3/'`
        if [ "$RECONCILE_COMPLETED" != 'True' ] && [ "$RECONCILE_COMPLETED" != 'False' ]; then
            echo "Error: ReconcileComplete not valid: $RECONCILE_COMPLETED_DATA"
            echo "Error: ReconcileComplete not valid: '${RECONCILE_COMPLETED}'"
            echo "HCO_DATA: $HCO_DATA"
            printf '%s' "$HCO_DATA" | hexdump -C
            printf '%s' "$RECONCILE_COMPLETED" | hexdump -C
            EXIT_STATUS=1
            SKIP_CHECK=1
            return
        fi
    fi

    # extract APPLICATION_AVAILABLE value : check that it has a valid value (either True or False)
    APPLICATION_AVAILABLE_DATA=`printf '%s' "$HCO_DATA" | sed -e 's/.*ApplicationAvailable: "\([^"]*\)" "\([^"]*\)" "\([^"]*\)".*$/Status: \1 Reason: \2 Message: \3/'`
    APPLICATION_AVAILABLE=`printf '%s' "$HCO_DATA" | sed -e 's/.*ApplicationAvailable: "\([^"]*\)".*$/\1/'`
    if [ "$APPLICATION_AVAILABLE" != 'True' ] && [ "$APPLICATION_AVAILABLE" != 'False' ]; then
        echo "Error: ApplicationAvailable not valid: $APPLICATION_AVAILABLE_DATA"
        echo "Error: ApplicationAvailable not valid: $APPLICATION_AVAILABLE"
        EXIT_STATUS=1
        SKIP_CHECK=1
        return
    fi
    
    # extract OPERATION_PROGRESSING value : check that it has a valid value (either True or False)
    OPERATION_PROGRESSING_DATA=`printf '%s' "$HCO_DATA" | sed -e 's/.*ApplicationProgressing: "\([^"]*\)" "\([^"]*\)" "\([^"]*\)".*$/Status: \1 Reason: \2 Message: \3/'`
    OPERATION_PROGRESSING=`printf '%s' "$HCO_DATA" | sed -e 's/.*ApplicationProgressing: "\([^"]*\)".*$/\1/'`
    if [ "$OPERATION_PROGRESSING" != 'True' ] && [ "$OPERATION_PROGRESSING" != 'False' ]; then
        echo "Error: OperationProgressing not valid: $OPERATION_PROGRESSING_DATA"
        echo "Error: OperationProgressing not valid: $OPERATION_PROGRESSING"
        EXIT_STATUS=1
        SKIP_CHECK=1
        return
    fi
    
    # extract APPLICATION_DEGRADED value : check that it has a valid value (either True or False)
    APPLICATION_DEGRADED_DATA=`printf '%s' "$HCO_DATA" | sed -e 's/.*ApplicationDegraded: "\([^"]*\)" "\([^"]*\)" "\([^"]*\)".*$/Status: \1 Reason: \2 Message: \3/'`
    APPLICATION_DEGRADED=`printf '%s' "$HCO_DATA" | sed -e 's/.*ApplicationDegraded: "\([^"]*\)".*$/\1/'`
    if [ "$APPLICATION_DEGRADED" != 'True' ] && [ "$APPLICATION_DEGRADED" != 'False' ]; then
        echo "Error: ApplicationDegraded not valid: $APPLICATION_DEGRADED_DATA"
        echo "Error: ApplicationDegraded not valid: $APPLICATION_DEGRADED"
        EXIT_STATUS=1
        SKIP_CHECK=1
        return
    fi
}


function check_operator_up() {
   local kind=$1
   local name=$2
   local namespace=$3

cat <<EOF

Checking operator kind: $kind name: $name 
EOF

   retry_count=0
   while [ "$retry_count" ]; do
      
      get_operator_json $kind $name $namespace $retry_count
      if [ "$SKIP_CHECK" == "1" ]; then
         SKIP_CHECK=0
         return
      fi
 
      # if reconcile status available: check that it is True (as a precondition for other checks)
      HAS_RECONCILE=""
      if [ "$has_reconcile" != "0" ]; then
         if [ "$RECONCILE_COMPLETED" == 'False' ] || [ "$RECONCILE_COMPLETED" == 'Unkown' ]; then
            ((retry_count=retry_count+1))
            echo "Waiting. Operator $kind :: reconcile not yet complete... status: $RECONCILE_COMPLETED (Retry ${retry_count}) "
            sleep 10
            continue
         fi
         HAS_RECONCILE="Reconcile completed and "
      fi

      # check APLICATION_AVAILABLE && APPLICATION_DEGRADED (subject to OPERATION_PROGRESSING)
      if [ "$APPLICATION_AVAILABLE" == 'True' ] && [ "$APPLICATION_DEGRADED" == 'False' ]; then
        if [ "$OPERATION_PROGRESSING" == 'False' ]; then
          echo "Success: Operator kind: $kind name: $name ${HAS_RECONCILE}is fully available"
          return
        fi
        REASON="Operator $kind available & not degraded, however still progressing"    
      else
        if [ "$OPERATION_PROGRESSING" == 'False' ]; then
            set +x
            if [ "$APPLICATION_AVAILABLE" == 'False' ]; then 
                echo "Error: Operator $kind is not is not available. Detailed status: $APPLICATION_AVAILABLE_DATA"
            fi
            if [ "$APPLICATION_DEGRADED" == 'True' ]; then 
                echo "Error: Operator $kind is degraded. Detailed status: $APPLICATION_DEGRADED_DATA"
            fi
            EXIT_STATUS=1
            return
        fi

        REASON=""
        if [ "$APPLICATION_DEGRADED" == 'True' ]; then 
            REASON="Operator $kind is degraded"
        fi
        if [ "$APPLICATION_AVAILABLE" == 'False' ]; then 
            REASON="Operator $kind is not available"
        fi
      fi

      ((retry_count=retry_count+1))
      echo "Waiting. Operator $kind :: $REASON - wait and retry as the operation is in progress (Retry ${retry_count})"
      sleep 10
   done

   echo "Error: timed out waiting for application to start."

   if [ "$has_reconcile" != "0" ] && [ "$RECONCILE_COMPLETED" == '"False"' ]; then
        echo "Reconcile not completed Extended information: ${RECONCILE_COMPLETED_DATA}"
   else
        if [ "$APPLICATION_AVAILABLE" == '"False"' ]; then 
            echo "Error: Operator $kind is not is not available. Detailed status: $APPLICATION_AVAILABLE_DATA"
        fi
        if [ "$APPLICATION_DEGRADED" == '"True"' ]; then 
            echo "Error: Operator $kind is degraded. Detailed status: $APPLICATION_DEGRADED_DATA"
        fi
   fi
   EXIT_STATUS=1
}

function check_dependent_operators_up {
    # get all operators mentioned as related objects of hco operator 
    RELATED_OBJECTS=`${CMD} get hyperconvergeds.hco.kubevirt.io kubevirt-hyperconverged -n kubevirt-hyperconverged -o go-template='{{range .status.relatedObjects }}{{if .namespace }}{{ printf "%s %s %s\n" .kind .name .namespace }}{{ else }}{{ printf "%s %s .\n" .kind .name }}{{ end }}{{ end }}'`

    # check that each operator is up
     while read line; do 

        fields=( $line )
        kind=${fields[0]} 
        name=${fields[1]} 
        namespace=${fields[2]} 

        if [ "$kind" != "ConfigMap" ]; then
            check_operator_up $kind $name $namespace
        fi
    done < <(echo "${RELATED_OBJECTS}")
}

echo "Waiting for operators to start. This check can take a long time..."

check_operator_up $HCO_KIND $HCO_NAME $HCO_NAMESPACE

#don't need to check if dependent operators are up; misunderstanding.
#check_dependent_operators_up

echo ""

if [ "$EXIT_STATUS" == "0" ]; then
    echo "Cluster is up and running. congratulations!"
else
    echo "Cluster is not up and running. Some of the operators are not ok."
    exit 1
fi


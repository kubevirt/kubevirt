#!/usr/bin/env bash
# Print the deployment order for a set of changed components.
set -eo pipefail

if [[ $# -lt 2 ]]; then
    echo "Usage: $0 MANIFEST COMPONENT [COMPONENT ...]" >&2
    exit 2
fi
manifest=$1
shift
changed=("$@")

changed_sorted=$(printf '%s\n' "${changed[@]}" | LC_ALL=C sort | paste -sd ' ' -)
rule_count=$(yq -r '.combination_flows // [] | length' "$manifest")
declare -A changed_lookup
for component in "${changed[@]}"; do
    COMPONENT=$component yq -e '.components[strenv(COMPONENT)] != null' "$manifest" >/dev/null || {
        echo "Unknown component '$component'" >&2
        exit 2
    }
    changed_lookup[$component]=1
done

matched_rule=-1
matched_exact=false
matched_specificity=-1
for ((rule_index = 0; rule_index < rule_count; rule_index++)); do
    if [[ $(yq -r ".combination_flows[$rule_index].exactly != null" "$manifest") == true ]]; then
        rule_components=$(yq -r ".combination_flows[$rule_index].exactly[]" "$manifest" | LC_ALL=C sort | paste -sd ' ' -)
        if [[ "$rule_components" == "$changed_sorted" ]]; then
            matched_rule=$rule_index
            matched_exact=true
            break
        fi
    elif [[ $(yq -r ".combination_flows[$rule_index].contains != null" "$manifest") == true ]]; then
        contains_components=()
        while IFS= read -r component; do
            [[ -z "$component" ]] || contains_components+=("$component")
        done < <(yq -r ".combination_flows[$rule_index].contains[]" "$manifest")
        contains_match=true
        for component in "${contains_components[@]}"; do
            if [[ -z "${changed_lookup[$component]:-}" ]]; then contains_match=false; break; fi
        done
        if [[ "$contains_match" == true && "$matched_exact" == false && ${#contains_components[@]} -gt $matched_specificity ]]; then
            matched_rule=$rule_index
            matched_specificity=${#contains_components[@]}
        fi
    fi
done

planned=()
if ((matched_rule >= 0)); then
    while IFS= read -r step; do
        if [[ "$step" == remaining_changed_components ]]; then
            mapfile -t rest < <(for component in "${changed[@]}"; do
                [[ " ${planned[*]} " == *" $component "* ]] && continue
                printf '%s\t%s\n' "$(COMPONENT=$component yq -er '.components[strenv(COMPONENT)].deploy_order' "$manifest")" "$component"
            done | sort -n | cut -f2)
            planned+=("${rest[@]}")
        elif [[ " ${changed[*]} " == *" $step "* ]]; then
            planned+=("$step")
        else
            echo "Combination flow references unchanged or unknown component '$step'" >&2
            exit 2
        fi
    done < <(yq -er ".combination_flows[$matched_rule].steps[]" "$manifest")
else
    mapfile -t planned < <(for component in "${changed[@]}"; do
        printf '%s\t%s\n' "$(COMPONENT=$component yq -er '.components[strenv(COMPONENT)].deploy_order' "$manifest")" "$component"
    done | sort -n | cut -f2)
fi

declare -A planned_lookup
for component in "${planned[@]}"; do
    if [[ -z "${changed_lookup[$component]:-}" || -n "${planned_lookup[$component]:-}" ]]; then
        echo "Combination flow contains an unchanged, unknown, or duplicate component '$component'" >&2
        exit 2
    fi
    planned_lookup[$component]=1
done
for component in "${changed[@]}"; do
    if [[ -z "${planned_lookup[$component]:-}" ]]; then
        echo "Combination flow omitted changed component '$component'" >&2
        exit 2
    fi
done
printf '%s\n' "${planned[@]}"

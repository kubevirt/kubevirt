#!/bin/bash

generate_random_number() {
    echo $((10000 + RANDOM % 90000))
}

declare -A num_hash
while IFS= read -r number; do
    num_hash["$number"]=1
done < <(find tests/ -name "*test.go" | xargs grep -oP '\[test_id:([0-9]+)\]' | grep -oP '\d+')

while IFS= read -r number; do
    if [ "${num_hash["$number"]}" ]; then
        echo "duplicate found $number"
        exit 1
    fi
    num_hash["$number"]=1
done < <(find tests/ -name "*test.go" | xargs grep -oP '\[test_cid:([0-9]+)\]' | grep -oP '\d+')

while IFS= read -r input_file; do
    temp_file=$(mktemp)
    while IFS= read -r line; do
        if [[ $line =~ 'Entry(' && ! $line =~ test_id && ! $line =~ test_cid ]]; then
            while true; do
                random_number=$(generate_random_number)
                if [ -z "${num_hash["$random_number"]}" ]; then
                    break
                fi
            done
            num_hash["$random_number"]=1
            line=$(echo "$line" | sed "s/Entry(\"/Entry(\"\[test_cid:$random_number]/")
        fi

        if [[ $line =~ 'It(' && ! $line =~ test_id && ! $line =~ test_cid ]]; then
            while true; do
                random_number=$(generate_random_number)
                if [ -z "${num_hash["$random_number"]}" ]; then
                    break
                fi
            done
            num_hash["$random_number"]=1
            line=$(echo "$line" | sed "s/It(\"/It(\"\[test_cid:$random_number]/")
        fi
        echo "$line" >> "$temp_file"
    done < "$input_file"
    mv "$temp_file" "$input_file"
done < <(find tests/ -name "*test.go")

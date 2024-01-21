#!/bin/bash

trigger_commit=$1
service_pattern=$2

all_services=( aaa chat e2e-test event frontend notification storage video )

echo "Service pattern: $service_pattern, commit: $trigger_commit"

website_prefix=https://chat.nkonev.name

force_run=$(git show -s --format=%s $trigger_commit | grep -q -F "[force]" && echo true || echo false)

services_list=()

if [[ "$force_run" == "true" ]]; then
  echo "Detected [force] message"
  services_list=("${all_services[@]}")
else
  echo "Going to determine changed dirs"
  parent_commits=()
  parent_commits=( $(git rev-parse $trigger_commit^@) )

  changed_dirs=()

  for service in "${all_services[@]}"; do
    echo "Examining service ${service}"
    if [[ "$service" == "frontend" ]]; then
      prev_deployed_commit=$(curl -Ss "$website_prefix/git.json" | jq -r '.commit')
    elif [[ "$service" == "e2e-test" ]]; then
      prev_deployed_commit=HEAD~1
    else
      prev_deployed_commit=$(curl -Ss "$website_prefix/${service}/git.json" | jq -r '.commit')
    fi

    echo "Getting change between ${prev_deployed_commit} and ${trigger_commit}"
    local_changed_dirs=( $(git diff --dirstat=files,0 ${prev_deployed_commit} ${trigger_commit} | sed 's/^[ 0-9.]\+% //g' | cut -d'/' -f1 | uniq) )
    echo "Since prev deployed commit ${prev_deployed_commit} there are following changed dirs"
    for changed_dir in "${local_changed_dirs[@]}"; do
        if [[ "$changed_dir" == "$service" ]]; then
          echo "-> ${changed_dir}"
          changed_dirs+=(${changed_dir})
        fi
    done

    echo
  done

  services_list=($(printf "%s\n" "${changed_dirs[@]}" | sort -u))
fi

echo "List of changed services: ${services_list[@]}"
#echo "${services_list[@]}"

echo "${services_list[@]}" | grep -E -q "${service_pattern}"

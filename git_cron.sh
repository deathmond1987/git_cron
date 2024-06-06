#!/usr/bin/env bash
set -xe
USER=deathmond1987
PROJECT_LIST=$(curl https://api.github.com/users/$USER/repos\?page\=1\&per_page\=100 | grep -e 'clone_url' | cut -d \" -f 4 | sed '/WSA/d' | xargs -L1)
GIT_DIR=~

cd "${GIT_DIR}"
for project in ${PROJECT_LIST}; do
    project_name=$(echo "${project}" | cut -d'/' -f 5)
    if [ -d ./"${project_name//.git/}" ]; then
        cd ./"${project_name//.git/}"
        git pull
        cd -
    else
        git clone ${project}
    fi
done

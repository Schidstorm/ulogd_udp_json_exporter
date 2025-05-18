#!/bin/bash

# user@host:port
remote=$1

user=$(echo $remote | cut -d'@' -f1)
host=$(echo $remote | cut -d'@' -f2 | cut -d':' -f1)
port=$(echo $remote | cut -d':' -f2)

if [[ -z $port ]]; then
    echo "Port not specified. exiting..."
    exit 1
fi

if [[ -z $user ]]; then
    echo "User not specified. exiting..."
    exit 1
fi
if [[ -z $host ]]; then
    echo "Host not specified. exiting..."
    exit 1
fi

if [[ -z $projectName ]]; then
    if [[ -f go.mod ]]; then
        projectName=$(grep -m 1 module go.mod | awk '{print $2}' | sed 's/.*\///')
    else
        projectName=$(basename $(pwd))
    fi
fi

loc() {
    echo "copy to remote"
    rsync -e "ssh -p $port" -avz --delete --exclude=.git --exclude=README.md "--exclude=${projectName}_*" . $user@$host:$projectName

    echo "cleanup binaries"
    ssh -p $port $user@$host 'cd '"$projectName"' && rm -f '"${projectName}_*"

    echo "build on remote"
    ssh -p $port $user@$host 'cd '"$projectName"' && bash remoteBuild.sh remote'

    echo "copy from remote"
    rsync -e "ssh -p $port" -avz "$user@$host:$projectName/${projectName}_*" .
}

remote() {
    make build_arch
}

cmd=$1

if [[ $cmd == "loc" ]]; then
    loc
elif [[ $cmd == "remote" ]]; then
    remote
else
    loc
fi


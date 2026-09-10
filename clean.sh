#!/bin/bash
docker stop $(docker ps -aq)
docker rm $(docker ps -aq)
echo "BISMILLAAAAAH"
read -p "Do you want to build? [y/N] " answer
if [[ "$answer" == "y" ]]; then
    docker build -f serverBuild/Dockerfile -t hugin-server .
fi
./huginn start configs/example.yaml

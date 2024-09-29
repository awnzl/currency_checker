#!/bin/bash

builder="currency_checker_builder"
logfile="build.log"

build_service() {
    local service_name=$1
    local dockerfile_path=$2

    echo "Building $service_name..."
    docker build --build-arg BUILDER=$builder -f $dockerfile_path -t "$service_name" >> $logfile 2>&1
    if [ $? -ne 0 ]; then
        echo "Error: Failed to build $service_name. Check $logfile for details."
        exit 1
    fi
}

# builder
docker build -f build.dockerfile -t $builder > $logfile 2>&1
if [ $? -ne 0 ]; then
    echo "Error: Failed to build builder image. Check $logfile for details."
    exit 1
fi

# services
build_service "currency_checker" "currency_checker/dockerfile"
build_service "price_collector" "price_collector/dockerfile"
build_service "rank_collector" "rank_collector/dockerfile"

echo "All builds completed successfully. See $logfile for the full log."

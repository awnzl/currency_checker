#!/bin/bash

SCRIPT_DIR=$(dirname "$(realpath "$0")")
BUILD_CONTEXT=$SCRIPT_DIR/..

builder="currency_checker_builder"
logfile="$SCRIPT_DIR/build.log"

echo "start building" > $logfile 2>&1

build_service() {
    local service_name=$1
    local dockerfile_path=$2

    build_cmd="docker build --build-arg BUILDER=$builder -f $dockerfile_path -t "$service_name" $BUILD_CONTEXT"
    echo "Building $service_name..." >> $logfile 2>&1
    echo "build command: '$build_cmd'" >> $logfile 2>&1
    docker build --build-arg BUILDER=$builder -f $dockerfile_path -t "$service_name" $SCRIPT_DIR >> $logfile 2>&1
    if [ $? -ne 0 ]; then
        echo "Error: Failed to build $service_name. Check $logfile for details."
        exit 1
    fi
}

# builder
build_cmd="docker build -f $SCRIPT_DIR/build.dockerfile -t $builder $BUILD_CONTEXT"
echo "build command: '$build_cmd'" >> $logfile 2>&1
$build_cmd >> $logfile 2>&1
if [ $? -ne 0 ]; then
    echo "Error: Failed to build builder image. Check $logfile for details."
    exit 1
fi

# services
build_service "currency_checker" $SCRIPT_DIR/"currency_checker/dockerfile"
build_service "price_collector" $SCRIPT_DIR/"price_collector/dockerfile"
build_service "rank_collector" $SCRIPT_DIR/"rank_collector/dockerfile"

docker image rm $builder

echo "All builds completed successfully. See $logfile for the full log."

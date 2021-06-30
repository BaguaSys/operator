#!/bin/bash
DIR=$(
    cd "$(dirname "$0")"
    pwd
)

function install() {
    for file in $(ls $DIR/crd); do
        kubectl apply -f $DIR/crd/$file
    done
}

function deploy() {
    install
    kubectl apply -f $DIR/deployment.yaml
}

deploy

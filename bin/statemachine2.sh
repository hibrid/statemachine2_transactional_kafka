#!/bin/bash -u
shopt -s extglob

DIR=$(cd $(dirname $(dirname ${0})); pwd)
MONGODB_SHELL=${MONGODB_SHELL:-`which mongosh`}

usage() {
  cat <<EOF
=======================================================
USAGE: statemachine2 command [service] [...additional args]

  ### Global Commands ###

  up                              Start the Kubernetes server and Statemachine2 services, in that order
  down                            Stop the Statemachine2 services and Kubernetes server, in that order

  destroy                         Shut down and completely remove the local Kubernetes server and
                                  all services


  ### Kubernetes Server Commands ###

  server-init                     Used to initialize or update the k8s stack. Specifically, it:
                                    1. Provisions and starts the Kubernetes server and registry
                                    2. Saves the Kubernetes server config to ${KUBECONFIG:-<undefined>}.

  server-start                    Start the Kind server and private registry containers
  server-stop                     Stop the Kind server and private registry containers


  ### Statemachine2 Services Commands ###

  start                           Provision and start the Statemachine2 services via Tilt
  stop                            Stop and remove the Statemachine2 services via Tilt
  restart [service]               Restart the specified service
  logs [service]                  Tail logs for the specified service

  exec service [...cmds]          Run shell commands in the currently running service
                                  container
                                    example:
                                      statemachine2 exec blip sh
                                      (to enter a shell inside the blip container)
                                    example:
                                      statemachine2 exec blip "ls -lR /app/node_modules/ | grep ^l | uniq"
                                      (to list all symlinked npm packages, such as after yarn linking)

  port-forward [service] [port]   Forward a port to a running service container

  restart-gateway                 Restart the gloo gateway services
                                  Useful if the exposed "blip" or "uploader" port stops responding on host machine

  restart-kafka                   Restart the kafka services


  ### Other Commands ###

  doctor                          Check to see if all dependancies are up-to-date, and recommended
                                  environment variables are set
EOF
}

server_init() {
  # Ensure docker is running before provisioning
  check_docker_running

  cd ${DIR} && ctlptl apply -f Kindconfig.yaml

}

server_start() {
  docker start statemachine-kind-control-plane
  docker start ctlptl-registry
}

server_stop() {
  docker stop statemachine-kind-control-plane
  docker stop ctlptl-registry
}

start() {
  cd ${DIR} && set_tilt_env ${1-} && tilt up -dv --port=0 --legacy
}

stop() {
  cd ${DIR} && set_tilt_env --shutdown && tilt down
}

up() {
  # Ensure docker is running before provisioning
  check_docker_running

  # Ensure mongodb replicaset connection before provisioning
  $MONGODB_SHELL --eval "rs.status()"

  server_start
  start
}

down() {
  stop
  server_stop
}

restart() { (for P in ${@}; do kubectl scale deployment --replicas=0 ${P} && kubectl scale deployment --replicas=1 ${P}; done) }

run_exec() { args="${@:2}" && (kubectl exec -ti svc/${1} -c ${1} -- /bin/sh -c "${args}") }

check_version() {
  pkg=${1}
  version_cmd=${2}
  min=${3}
  extra=${4}
  if [ $([[ $($pkg $version_cmd $extra) =~ [0-9\.]+ ]] && printf "${min}\n${BASH_REMATCH}" | sort -V | head -n1) = "$min" ]; then
      echo "✔ ${pkg} is up to date"
  else
      echo "✘ ${pkg} is out of date. Please install version >= ${min}"
  fi
}

check_docker_running() {
  # Ensure docker is running
  if (! docker stats --no-stream ); then
    echo "Docker is not running"
    exit 0;
  fi
}

check_environment() {
  if [[ ! -z ${!1} ]]; then
    echo "✔ \$${1} is set"
  else
    echo "✘ \$${1} is not set. See \"Environment Setup\" in README"
  fi
}

doctor() {
  set +u
  echo "Checking environment config..."
  check_environment KUBECONFIG
  echo
  echo "Checking dependancy versions..."
  check_version ctlptl 'version' 0.8.5
  check_version helm 'version --short' 3.9.1
  check_version tilt 'version' 0.33.5
  check_version docker '-v' 20.10.17
  check_version kubectl "version -oyaml --client" 1.21.12 "|awk '/gitVersion/{print $2;}'"
}

set_tilt_env() {
  case "${1-}" in
    --trap-shutdown) trap 'SHUTTING_DOWN=1 tilt down' EXIT;;
    --shutdown) export SHUTTING_DOWN=1;;
    *) ;;
  esac
}

wait_for_kubernetes_ready() {
  until curl -s --fail http://127.0.0.1:10080/kubernetes-ready; do
    sleep 1;
  done
}

destroy() {
  read -p "Are you sure you want to destroy the local statemachine server? [N|y] " -n 1 -r
  echo
  if [[ $REPLY =~ ^[Yy]$ ]]
  then
    docker rm -fv statemachine-kind-control-plane
    docker rm -fv ctlptl-registry
  else
    echo "Destroy command aborted"
  fi
}

case ${1-help} in
  up) up;;
  down) down;;
  destroy) destroy;;
  server-init) server_init;;
  server-start) server_start;;
  server-stop) server_stop;;
  start) start ${2-"--trap-shutdown"};;
  stop) stop;;
  restart) shift 1 && restart ${@};;
  logs) (cd ${DIR} && kubectl logs svc/${2} --tail=100 -c ${2} -f);;
  exec) run_exec ${2} ${@:3};;
  port-forward) (cd ${DIR} && kubectl port-forward svc/${2} ${3});;
  restart-gateway) restart gloo gateway-proxy gateway discovery;;
  restart-kafka) restart default-kafka-connect-mongo-connect kafka-entity-operator;;
  doctor) doctor;;
  *) usage;;
esac
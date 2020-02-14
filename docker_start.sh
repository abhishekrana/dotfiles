#!/bin/bash

REPO_NAME="ubuntu"
REPO_VERSION="18.04"

CONTAINER_NAME="${REPO_NAME//[\/]/_}"_"$REPO_VERSION"_"$USER"
CONTAINER=$CONTAINER_NAME

docker_run()
{
	# Docker versions earlier than 19.03 require nvidia-docker2 and the --runtime=nvidia flag.
	# On versions including and after 19.03, you will use the nvidia-container-toolkit package and the --gpus all flag
	GPU_FLAG=""
	echo ""
	lspci | grep -i nvidia
	if [ $? -eq 0 ]; then
		GPU_FLAG="--gpus all"
	fi
	echo "GPU_FLAG: $GPU_FLAG"
	echo ""

	docker run $GPU_FLAG -d -it \
		-e CURL_CA_BUNDLE=$CURL_CA_BUNDLE \
		-e GS_ACCESS_KEY_ID=$GS_ACCESS_KEY_ID \
		-e GS_SECRET_ACCESS_KEY=$GS_SECRET_ACCESS_KEY \
		-e DISPLAY=$DISPLAY \
		-e TMUX=$TMUX \
		-v /tmp/.X11-unix:/tmp/.X11-unix \
		-v /var/run/docker.sock:/var/run/docker.sock \
		-v ~/.config/pudb:/root/.config/pudb \
		-v ~/aSk/dotfiles:/root/aSk/dotfiles \
		-v ~/aSk/dotfiles/nvim/coc-settings.json:/root/.config/nvim/coc-settings.json \
		-v /home:/home \
		-v /mnt:/mnt \
		-v /media:/media \
		--net=host \
		--name $CONTAINER_NAME \
		"$REPO_NAME":"$REPO_VERSION" \
		bash
		# bash -c "pip install pudb && cd `pwd` && bash"
	echo "CREATED - $CONTAINER_NAME"
}

docker_exec()
{
	STARTED=$(docker inspect --format="{{.State.StartedAt}}" $CONTAINER_NAME)
	NETWORK=$(docker inspect --format="{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}" $CONTAINER_NAME)
	echo ""
	echo "OK - $CONTAINER_NAME is running. IP: $NETWORK, StartedAt: $STARTED"
	echo ""
	docker exec -it $CONTAINER_NAME bash -c "cd `pwd` && bash"
}

##### MAIN #####

for i in "$@"
do
case $i in
    -b=*|--build=*)
    ARG_BUILD="${i#*=}"
    shift # past argument=value
    ;;
	-c=*|--container-restart=*)
    ARG_CONTAINER_RESTART="${i#*=}"
    shift # past argument=value
    ;;
    -p=*|--pull=*)
    ARG_PULL="${i#*=}"
    shift # past argument=value
    ;;
	-h=*|--help=*)
    ARG_PULL="${i#*=}"
    shift # past argument=value
	echo "-p=1: Pull docker image. Create container (if it does not exist). SSH into the container (default)"
	echo "-b=1: Build docker image using Dockerfile. Create container (if it does not exist). SSH into the container"
	echo "-c=1: Stop and remove container. Create container. SSH into the container"
	exit 0
    ;;
    --default)
    ARG_DEFAULT=YES
    shift # past argument with no value
    ;;
    *)
		echo "ERROR:: INVALID ARGUMENT"
		exit 1
    ;;
esac
done

## Check docker version (to support docker run --gpus all)
docker_cur_ver="$( docker --version | cut -d ' ' -f 3 | cut -d ',' -f 1 )"
docker_req_ver="19.03"
if [ "$(printf '%s\n' "$docker_req_ver" "$docker_cur_ver" | sort -V | head -n1)" = "$docker_req_ver" ];then
	echo "Docker version: $docker_cur_ver"
	echo ""
else
	echo "ERROR:: Docker version >= $docker_req_ver required"
	exit 1
fi

## Build docker image
if [[ "$ARG_BUILD" == "1" ]];then
	docker build -t "$REPO_NAME":"$REPO_VERSION" -f Dockerfile .
fi

## Pull docker image
if [[ "$ARG_PULL" == "1" ]];then
	docker pull "$REPO_NAME":"$REPO_VERSION"
fi

## Pull docker image if it does not exist
if [[ "$(docker images -q $REPO_NAME:$REPO_VERSION 2> /dev/null)" == "" ]]; then
	docker pull "$REPO_NAME":"$REPO_VERSION"
fi

## Create docker container and ssh into it
RUNNING=$(docker inspect --format="{{.State.Running}}" $CONTAINER_NAME 2> /dev/null)
if [ $? -eq 1 ]; then
	echo "UNKNOWN - $CONTAINER_NAME does not exist."
	docker_run
	docker_exec
fi
if [ "$RUNNING" == "false" ];then
	echo "CRITICAL - $CONTAINER_NAME is not running."
	docker rm $CONTAINER_NAME
	docker_run
	docker_exec
elif [ "$RUNNING" == "true" ];then
	if [[ "$ARG_CONTAINER_RESTART" == "1" ]];then
		echo "STOPPING $CONTAINER_NAME."
		docker stop $CONTAINER_NAME > /dev/null
		echo "REMOVING $CONTAINER_NAME."
		docker rm $CONTAINER_NAME > /dev/null
		echo "CREATING $CONTAINER_NAME."
		docker_run
		docker_exec
	else
		echo "SSH into $CONTAINER_NAME"
		docker_exec
	fi
fi

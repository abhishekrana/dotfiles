REPO_NAME="dotfiles"
REPO_VERSION="1.0"

CONTAINER_NAME="${REPO_NAME//[\/]/__}"__"cnt1"
PWD_NAME=` pwd | awk -F'/' '{print $NF}' `

echo $CONTAINER_NAME
echo $PWD_NAME

CURL_CA_BUNDLE=""
GS_ACCESS_KEY_ID=""
GS_SECRET_ACCESS_KEY=""


## Build docker image
# docker build -t "$REPO_NAME":"$REPO_VERSION" .

## Stop and remove container
# docker stop $CONTAINER_NAME
# docker rm $CONTAINER_NAME


nvidia-smi
if [[ $? == 0 ]];then
	sudo docker run -it \
		--ipc=host \
		--runtime=nvidia \
		-e DISPLAY=$DISPLAY \
        -e CURL_CA_BUNDLE=$CURL_CA_BUNDLE \
		-e GS_ACCESS_KEY_ID=$GS_ACCESS_KEY_ID \
		-e GS_SECRET_ACCESS_KEY=$GS_SECRET_ACCESS_KEY \
		-e GDAL_DISABLE_READDIR_ON_OPEN=YES \
		-e CPL_VSIL_CURL_ALLOWED_EXTENSIONS=.tif \
		-v /tmp/.X11-unix:/tmp/.X11-unix \
		-v ~/.config/pudb:/root/.config/pudb \
		-v /home:/home \
		-v /mnt:/mnt \
		--name $CONTAINER_NAME \
		"$REPO_NAME":"$REPO_VERSION" \
		bash -c "pip install pudb gcloud && pip install --upgrade Pillow shapely geopandas && pip install --upgrade tensorflow-gpu && apt install -y vim && cd `pwd` && bash"
else
	sudo docker run -it \
		--ipc=host \
		-e DISPLAY=$DISPLAY \
        -e CURL_CA_BUNDLE=$CURL_CA_BUNDLE \
		-e GS_ACCESS_KEY_ID=$GS_ACCESS_KEY_ID \
		-e GS_SECRET_ACCESS_KEY=$GS_SECRET_ACCESS_KEY \
		-e GDAL_DISABLE_READDIR_ON_OPEN=YES \
		-e CPL_VSIL_CURL_ALLOWED_EXTENSIONS=.tif \
		-v /tmp/.X11-unix:/tmp/.X11-unix \
		-v /home:/home \
		-v /mnt:/mnt \
		-v ~/.config/pudb:/root/.config/pudb \
		-v ~/.config/nvim:/root/.config/nvim \
		-v ~/.config/coc:/root/.config/coc \
		-v ~/aSk:/root/aSk \
		--name $CONTAINER_NAME \
		"$REPO_NAME":"$REPO_VERSION" \
		bash -c "echo source ~/aSk/dotfiles/bash/.bashrc_aSk >> /root/.bashrc && cd `pwd` && bash"
fi


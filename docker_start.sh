REPO_NAME="dotfiles"
REPO_VERSION="1.0"

CONTAINER_NAME="${REPO_NAME//[\/]/__}"__"cnt1"
PWD_NAME=` pwd | awk -F'/' '{print $NF}' `

echo $CONTAINER_NAME
echo $PWD_NAME

rm -f .bash_history .viminfo

# Build docker image
docker build -t "$REPO_NAME":"$REPO_VERSION" .

# Stop and remove container
docker stop $CONTAINER_NAME
docker rm $CONTAINER_NAME

# Create container
xhost +
docker run -dit \
	-e DISPLAY=$DISPLAY \
	--ipc=host \
	-v /tmp/.X11-unix:/tmp/.X11-unix \
	-v `pwd`:/workspace/$PWD_NAME \
	# -v /home/abhishek/Abhishek/git/dotfiles:/workspace/dotfiles \
	--name $CONTAINER_NAME \
	"$REPO_NAME":"$REPO_VERSION"

# Start container
docker attach "$CONTAINER_NAME"


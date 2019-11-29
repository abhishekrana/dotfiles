ROOT_PATH="$HOME/aSk"
mkdir -p $ROOT_PATH

## Vim key binding in terminal
wget https://raw.githubusercontent.com/abhishekrana/dotfiles/master/misc/.inputrc -O ~/.inputrc

## Install packages
sudo apt-get update
sudo apt-get install -y python-autopep8 cmake build-essential universal-ctags silversearcher-ag



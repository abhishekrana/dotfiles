#!/bin/bash

#NEOVIM_VERSION="v0.4.3"
NEOVIM_VERSION="nightly"
ROOT_PATH="$HOME/aSk"

apt-get update
apt-get install -y sudo
sudo apt-get update

## Misc
sudo apt-get install -y python-autopep8 universal-ctags silversearcher-ag xclip xsel

## Prerequisites
mkdir -p $ROOT_PATH $ROOT_PATH/bin $ROOT_PATH/lib
sudo apt install -y python-pip python3-pip curl
## Install Neovim
if ! [ -x "$(command -v nvim)" ];then
	pip install --upgrade pip
	pip3 install --upgrade neovim
	pip3 install --user --upgrade neovim	# Install without no root access
	curl -LO https://github.com/neovim/neovim/releases/download/${NEOVIM_VERSION}/nvim.appimage
	chmod u+x nvim.appimage
	./nvim.appimage --appimage-extract
	cp -r squashfs-root/usr/* ~/aSk/
	rm -rf squashfs-root
	rm nvim.appimage
	cp ~/aSk/bin/nvim ~/aSk/bin/vim
else
	echo "nvim already installed"
fi


# Install Neovim plugin dependencies
pip install --upgrade jedi setuptools pylint python-language-server[all] pynvim
pip3 install --upgrade jedi setuptools pylint python-language-server[all] pynvim
curl -sL install-node.now.sh/lts | sudo bash


### Setup Neovim config
cp nvim/init_aSk.vim ~/aSk/
mkdir -p ~/.config/nvim

grep -q "init_aSk" "~/.config/nvim/init.vim"
if [[ $? != 0 ]];then
	curl -fLo ~/.local/share/nvim/site/autoload/plug.vim --create-dirs \
		https://raw.githubusercontent.com/junegunn/vim-plug/master/plug.vim
	mkdir -p ~/.config/nvim
	echo '' >> ~/.config/nvim/init.vim
	echo '""" aSk'  >> ~/.config/nvim/init.vim
	echo 'so ~/aSk/init_aSk.vim' >> ~/.config/nvim/init.vim
	~/aSk/bin/nvim +PlugInstall +qall
	echo "~/.config/nvim/init.vim updated"
else
	echo "~/.config/nvim/init.vim not updated"
fi

sudo npm install -g neovim

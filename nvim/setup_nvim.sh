#!/bin/bash 

## Download neovim
curl -LO https://github.com/neovim/neovim/releases/download/stable/nvim.appimage
chmod u+x nvim.appimage
sudo mv nvim.appimage /usr/bin/nvim

## Setup alias
grep -q "alias vim" ~/.bashrc
if [[ $? != 0 ]];then
    echo 'alias vim='/usr/bin/nvim'' >> ~/.bashrc
    echo "~/.bashrc updated"
fi

## Setup config
mkdir -p ~/.config/nvim
cp init.vim ~/.config/nvim/
cp coc-settings.json ~/.config/nvim/

## Install dependencies
sudo apt-get install -y python-autopep8 ctags silversearcher-ag xclip xsel
pip install --upgrade jedi setuptools pylint python-language-server[all] pynvim
pip3 install --upgrade jedi setuptools pylint python-language-server[all] pynvim
curl -sL install-node.now.sh/lts | sudo bash
sudo npm install -g neovim

## Install plugins
/usr/bin/nvim +PlugInstall +qall

/usr/bin/nvim --version

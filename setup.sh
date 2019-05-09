#/bin/bash

### Globals
NEOVIM_VERSION="nightly"
TMUX_VERSION=2.9

ROOT_PATH="$HOME/aSk"
ROOT_ACCESS=0


# while true; do
#     read -p "Do you have root access? " yn
#     case $yn in
#         [Yy]* ) 
# 			if ! [ -x "$(command -v sudo)" ];then
# 				apt update
# 				apt install -y sudo
# 			fi
# 			sudo who
# 			ROOT_ACCESS=1;
# 			break;;
#         [Nn]* ) 
# 			ROOT_ACCESS=0 
# 			break;;
#         * ) 
# 			echo "Please answer yes or no.";;
#     esac
# done
# echo "ROOT_ACCESS: $ROOT_ACCESS"


# ### Setup
mkdir -p $ROOT_PATH $ROOT_PATH/bin $ROOT_PATH/lib


# ### Get dotfiles
#if [ ! -d $ROOT_PATH/dotfiles ]; then
#	# git clone https://github.com/abhishekrana/dotfiles $ROOT_PATH/dotfiles
#	cp -r ../dotfiles $ROOT_PATH/dotfiles 
#fi


# ### Install Neovim
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


# ### Setup bashrc config
grep -q "bashrc_aSk" ~/.bashrc
if [[ $? != 0 ]];then
	echo '' >> ~/.bashrc
	echo '### aSk'  >> ~/.bashrc
	echo 'if [ -f ~/aSk/dotfiles/bash/.bashrc_aSk ]; then
		source ~/aSk/dotfiles/bash/.bashrc_aSk
	fi' >> ~/.bashrc
	echo "~/.bashrc updated"
else
	echo "~/.bashrc not updated"
fi


# ### Install Neovim plugin dependencies
# pip install --upgrade jedi
# pip install --upgrade setuptools
pip install --upgrade --user pylint
pip3 install --upgrade --user pylint
pip install --upgrade --user python-language-server[all]
pip3 install --upgrade --user python-language-server[all]
pip3 install --user pynvim # Denite


### Setup Neovim config
grep -q "init_aSk" "~/.config/nvim/init.vim"
if [[ $? != 0 ]];then
	curl -fLo ~/.local/share/nvim/site/autoload/plug.vim --create-dirs \
		https://raw.githubusercontent.com/junegunn/vim-plug/master/plug.vim
	mkdir -p ~/.config/nvim
	echo '' >> ~/.config/nvim/init.vim
	echo '""" aSk'  >> ~/.config/nvim/init.vim
	echo 'so ~/aSk/dotfiles/neovim/init_aSk.vim' >> ~/.config/nvim/init.vim
	~/aSk/bin/nvim +PlugInstall +qall
	echo "~/.config/nvim/init.vim updated"
else
	echo "~/.config/nvim/init.vim not updated"
fi


### Install Tmux
git clone https://github.com/tmux-plugins/tpm ~/.tmux/plugins/tpm
wget https://raw.githubusercontent.com/abhishekrana/dotfiles/master/tmux/.tmux.conf -O ~/.tmux.conf
# <leader_key> I
if [ "$EUID" -eq 0 ];then
	# Root user
	apt-get -y install sudo
	sudo apt-get -y install wget tar libevent-dev libncurses-dev
	cd /tmp
	wget https://github.com/tmux/tmux/releases/download/${TMUX_VERSION}/tmux-${TMUX_VERSION}.tar.gz
	tar xf tmux-${TMUX_VERSION}.tar.gz
	rm -f tmux-${TMUX_VERSION}.tar.gz
	cd tmux-${TMUX_VERSION}
	./configure
	make
	sudo make install
	cd -
	sudo rm -rf /usr/local/src/tmux-\*
	sudo mv tmux-${TMUX_VERSION} /usr/local/src
fi


### Vim key binding in terminal
wget https://raw.githubusercontent.com/abhishekrana/dotfiles/master/misc/.inputrc -O ~/.inputrc


### Install ack
# if ! [ -x "$(command -v ack)" ];then
# 	# curl https://beyondgrep.com/ack-2.24-single-file > ~/bin/ack && chmod 0755 ~/bin/ack 
# 	curl https://beyondgrep.com/ack-2.24-single-file > ~/aSk/bin/ack && chmod 0755 ~/aSk/bin/ack 
# else
# 	echo "ack already installed"
# fi

# if ! [ -x "$(command -v ag)" ];then
# 	apt install -y silversearcher-ag
# else
# 	echo "ag already installed"
# fi


### Install dependencies
echo ""
echo "##########"
if [ "$EUID" -eq 0 ];then
  # Root user
  sudo apt-get update
  sudo apt-get install -y python-autopep8 cmake build-essential exuberant-ctags
else
  echo "Manually install:"
  echo "sudo apt-get install -y python-autopep8 cmake build-essential exuberant-ctags"
fi

echo "Manually install tmux plugins: <leader_key> I"
echo ""


# Update tmux to 2.9
# vim -
# :checkhealth
# :CocInstall coc-python



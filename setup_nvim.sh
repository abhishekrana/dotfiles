NEOVIM_VERSION="v0.4.3"
ROOT_PATH="$HOME/aSk"

sudo apt-get update

## Prerequisites
mkdir -p $ROOT_PATH $ROOT_PATH/bin $ROOT_PATH/lib
sudo apt install python-pip
sudo apt install python3-pip
sudo apt install curl
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
pip install --upgrade jedi
pip install --upgrade setuptools
pip install --upgrade --user pylint
pip3 install --upgrade --user pylint
pip install --upgrade --user python-language-server[all]
pip3 install --upgrade --user python-language-server[all]
pip3 install --user pynvim


### Setup Neovim config
grep -q "init_aSk" "~/.config/nvim/init.vim"
cp neovim/init_aSk.vim ~/aSk/
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


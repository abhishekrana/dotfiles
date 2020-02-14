#!/bin/bash

install_pkgs()
{
	# apt-get update
	apt-get install -y \
		sudo \
		python3-pip \
		wget \
		git \
		exuberant-ctags \
		locate \
		curl \
		unzip \
		tree \
		htop \
		file \
		dialog \
		silversearcher-ag \
		ranger
		# vim
}

install_pip_pkgs()
{
	# apt update && apt install python3-pip
	pip3 install --upgrade pip
	pip3 install \
		google-api-python-client \
		oauth2client \
		setuptools \
		jupyter \
		ipython \
		matplotlib \
		dotted-dict \
		click \
		tqdm \
		colorlog \
		typeguard \
		mypy \
		monkeytype \
		black
}

install_tensorflow_intellisence()
{
	# For TensorFlow IntelliSence
	PYTHON_VERSION=$(python3 -V | cut -d " " -f 2 | cut -d "." -f 1-2)
	mkdir -p /root/.local/virtual-site-packages \
		&& ln -s /usr/local/lib/python{$PYTHON_VERSION}/dist-packages/tensorflow_core /root/.local/virtual-site-packages/tensorflow

	# For Tensorflow linting (pylint)
	grep -qxF 'export PYTHONPATH=$PYTHONPATH:/root/.local/virtual-site-packages # aSk' /root/.bashrc || echo 'export PYTHONPATH=$PYTHONPATH:/root/.local/virtual-site-packages # aSk' >> /root/.bashrc
}

install_pudb()
{
	pip3 --no-cache-dir install pudb
	# apt-get update && apt-get install -y locales
	apt-get install -y locales
	locale-gen en_US.UTF-8
}

install_neovim()
{
	PWD_PATH=`pwd`

	# Update .bashrc
	grep -qxF 'alias vim="/root/squashfs-root/usr/bin/nvim" # aSk' /root/.bashrc || echo 'alias vim="/root/squashfs-root/usr/bin/nvim" # aSk' >> /root/.bashrc
	grep -qxF 'export TERM=screen-256color # aSk' /root/.bashrc || echo 'export TERM=screen-256color # aSk' >> /root/.bashrc

	# Install neovim
	cd /root \
		&& wget https://github.com/neovim/neovim/releases/download/nightly/nvim.appimage \
		&& chmod u+x nvim.appimage \
		&& ./nvim.appimage --appimage-extract
	cd $PWD_PATH

	# Install plugin dependencies
	curl -sL install-node.now.sh/lts > node.sh; chmod +x node.sh; ./node.sh --yes; rm node.sh
	curl -LO https://github.com/BurntSushi/ripgrep/releases/download/11.0.2/ripgrep_11.0.2_amd64.deb \
		&& sudo dpkg -i ripgrep_11.0.2_amd64.deb \
		&& rm ripgrep_11.0.2_amd64.deb 
	# apt-get update && apt-get install -y silversearcher-ag
	apt-get install -y silversearcher-ag
	pip3 install python-language-server[all] jedi pynvim pylint black pyls-black pyls-mypy pyls-isort ptpython
	npm install -g neovim

	# Setup configuration files
	mkdir -p /root/.config/nvim
	cp -v .vim/coc-settings.json /root/.config/nvim/coc-settings.json
	cp -v .vim/init.vim /root/.config/nvim/init.vim

	# Install neovim plugins
	/root/squashfs-root/usr/bin/nvim -E -s -u /root/.config/nvim/init.vim +PlugInstall +qall 2>&1
}


setup_configs()
{
	STR="if [ -f ~/aSk/dotfiles/bash/bashrc ]; then source ~/aSk/dotfiles/bash/bashrc; fi # aSk"
	grep -qxF "$STR" /root/.bashrc || echo "$STR" >> /root/.bashrc

	grep -qxF 'so ~/aSk/dotfiles/nvim/init.vim " aSk' /root/.config/nvim/init.vim || echo 'so ~/aSk/dotfiles/nvim/init.vim " aSk' >> /root/.config/nvim/init.vim

}
##### MAIN #####
# Prompt for password if not root user
if [ $EUID != 0 ]; then
    sudo "$0" "$@"
    exit $?
fi

apt-get update
install_pkgs
install_pip_pkgs
install_tensorflow_intellisence
install_pudb
install_neovim
setup_configs
bash


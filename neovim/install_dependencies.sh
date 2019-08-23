#!/usr/bin/env bash

# https://github.com/neovim/neovim/wiki/Installing-Neovim 

PIP=`which pip`
PIP2=`which pip2`
PIP3=`which pip3`
echo $PIP
echo $PIP2
echo $PIP3

sudo apt update

# while getopts u:d:p:f: option
# do
# case "${option}"
# in
# u) USER=${OPTARG};;
# d) DATE=${OPTARG};;
# p) PRODUCT=${OPTARG};;
# f) FORMAT=${OPTARG};;
# esac
# done

## Dependencies
sudo $PIP install --upgrade pip
sudo $PIP2 install --upgrade pip2
sudo $PIP3 install --upgrade pip

sudo $PIP install --upgrade pynvim pylint neovim
sudo $PIP2 install --upgrade pynvim pylint neovim
sudo $PIP3 install --upgrade pynvim pylint neovim

sudo npm install -g neovim
npm install -g neovim

sudo gem install neovim

# sudo apt-get install python-neovim
# sudo apt-get install python3-neovim


### coc.vim ###
set -o nounset    # error when referencing undefined variable
set -o errexit    # exit when command fails

# Install latest nodejs
if [ ! -x "$(command -v node)" ]; then
    # curl --fail -LSs https://install-node.now.sh/latest | sh
	cd /tmp
    curl --fail -LSs https://install-node.now.sh/latest > nodejs.sh
	chmod +x nodejs.sh 
	./nodejs.sh --prefix=$HOME/aSk -y
	rm nodejs.sh

	# Install yarn if you want to build coc.nvim from source code.
	# curl --compressed -o- -L https://yarnpkg.com/install.sh | bash

    export PATH="$HOME/aSk/bin/:$PATH"
    # Or use apt-get
    # sudo apt-get install nodejs
fi

# Use package feature to install coc.nvim
# mkdir -p ~/.local/share/nvim/site/pack/coc/start
# cd ~/.local/share/nvim/site/pack/coc/start
# curl --fail -L https://github.com/neoclide/coc.nvim/archive/release.tar.gz|tar xzfv -

# Install extensions
mkdir -p ~/.config/coc/extensions
cd ~/.config/coc/extensions
if [ ! -f package.json ]
then
  echo '{"dependencies":{}}'> package.json
fi
# Change extension names to the extensions you need
npm install coc-json --global-style --ignore-scripts --no-bin-links --no-package-lock --only=prod
npm install coc-python --global-style --ignore-scripts --no-bin-links --no-package-lock --only=prod
npm install coc-pairs --global-style --ignore-scripts --no-bin-links --no-package-lock --only=prod
npm install coc-lists --global-style --ignore-scripts --no-bin-links --no-package-lock --only=prod



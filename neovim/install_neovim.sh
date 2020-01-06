NEOVIM_VERSION="nightly" # "0.5.0"

## Install Neovim
echo "Installing nvim ..."
cd /tmp
curl -LO https://github.com/neovim/neovim/releases/download/${NEOVIM_VERSION}/nvim.appimage
chmod u+x nvim.appimage
./nvim.appimage --appimage-extract
mkdir -p ~/aSk
cp -r squashfs-root/usr/* ~/aSk/
rm -rf squashfs-root
rm nvim.appimage
cp ~/aSk/bin/nvim ~/aSk/bin/vim
echo "nvim installed"

pip3 install --upgrade pip
pip3 install --upgrade neovim
pip3 install --user --upgrade neovim	# Install without no root access

# echo "alias vim='/root/aSk/squashfs-root/usr/bin/nvim'" >> /root/.bashrc
# echo "export TERM=screen-256color" >> /root/.bashrc

# Install dependenies
curl -sL install-node.now.sh/lts > node.sh; chmod +x node.sh; sudo ./node.sh --yes; rm node.sh

curl -LO https://github.com/BurntSushi/ripgrep/releases/download/11.0.2/ripgrep_11.0.2_amd64.deb && \
sudo dpkg -i ripgrep_11.0.2_amd64.deb && \
rm ripgrep_11.0.2_amd64.deb

sudo apt-get update && sudo apt-get install -y silversearcher-ag xsel

pip3 --no-cache-dir install python-language-server[all] pynvim pylint black pyls-black pyls-mypy pyls-isort

sudo npm install -g neovim

# cp .vim/coc-settings.json ~/.config/nvim/coc-settings.json
# cp .vim/init.vim ~/.config/nvim/init.vim

# nvim -E -s -u ~/.config/nvim/init.vim +PlugInstall +qall 2>&1

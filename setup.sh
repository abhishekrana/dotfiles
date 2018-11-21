#/bin/bash

mkdir -p /tmp/tmp_1
cd /tmp/tmp_1
mkdir -p ~/aSk


## Vim
# git clone https://github.com/VundleVim/Vundle.vim.git ~/.vim/bundle/Vundle.vim
# wget https://raw.githubusercontent.com/abhishekrana/dotfiles/master/.vimrc -O ~/.vimrc
# vim +BundleInstall +qall


## Neovim
pip install --upgrade pip
pip3 install --upgrade neovim
pip3 install --user --upgrade neovim	# Install without no root access
curl -LO https://github.com/neovim/neovim/releases/download/v0.3.1/nvim.appimage
chmod u+x nvim.appimage
# ./nvim.appimage
./nvim.appimage --appimage-extract
cp -r squashfs-root/usr/* ~/aSk/
cp ~/aSk/bin/nvim ~/aSk/bin/vim

echo 'export PATH='$PATH':~/aSk/bin' >> ~/.bashrc

# Neovim config and plugins
curl -fLo ~/.local/share/nvim/site/autoload/plug.vim --create-dirs \
    https://raw.githubusercontent.com/junegunn/vim-plug/master/plug.vim
mkdir -p ~/.config/nvim
wget https://raw.githubusercontent.com/abhishekrana/dotfiles/master/neovim/init.vim -O ~/.config/nvim/init.vim
~/aSk/bin/nvim +PlugInstall +qall


## Tmux
git clone https://github.com/tmux-plugins/tpm ~/.tmux/plugins/tpm
wget https://raw.githubusercontent.com/abhishekrana/dotfiles/master/tmux/.tmux.conf -O ~/.tmux.conf
# <leader_key> I


## Vim key binding in terminal
wget https://raw.githubusercontent.com/abhishekrana/dotfiles/master/misc/.inputrc -O ~/.inputrc


## Dependencies
# apt install exuberant-ctags


## Cleanup
cd ~;rm -rf /tmp/tmp_1

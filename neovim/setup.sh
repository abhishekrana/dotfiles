
pip3 install --user neovim

pip3 install neovim
pip3 install --upgrade neovim

curl -LO https://github.com/neovim/neovim/releases/download/v0.3.1/nvim.appimage
chmod u+x nvim.appimage
#./nvim.appimage

mkdir ~/.config/nvim/

cd ~/.config/nvim
git clone https://github.com/VundleVim/Vundle.vim.git ~/.config/nvim/bundle/Vundle.vim

# dotfiles


# Vim
git clone https://github.com/VundleVim/Vundle.vim.git ~/.vim/bundle/Vundle.vim
wget https://raw.githubusercontent.com/abhishekrana/dotfiles/master/.vimrc -O ~/.vimrc
vim +BundleInstall +qall

# Tmux
git clone https://github.com/tmux-plugins/tpm ~/.tmux/plugins/tpm
wget https://raw.githubusercontent.com/abhishekrana/dotfiles/master/.tmux.conf -O ~/.tmux.conf
<leader_key> I

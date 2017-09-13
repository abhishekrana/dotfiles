# dotfiles

![Screenshot](screenshot1.png)

# Vim
git clone https://github.com/VundleVim/Vundle.vim.git ~/.vim/bundle/Vundle.vim

wget https://raw.githubusercontent.com/abhishekrana/dotfiles/master/.vimrc -O ~/.vimrc

vim +BundleInstall +qall


# Tmux
git clone https://github.com/tmux-plugins/tpm ~/.tmux/plugins/tpm

wget https://raw.githubusercontent.com/abhishekrana/dotfiles/master/.tmux.conf -O ~/.tmux.conf

<leader_key> I


# YouCompleteMe
sudo apt-get install build-essential cmake

sudo apt-get install python-dev python3-dev

cd ~/.vim/bundle/YouCompleteMe

./install.py --clang-completer


# Vim key binding in terminal
wget https://raw.githubusercontent.com/abhishekrana/dotfiles/master/.inputrc -O ~/.inputrc


# Terminator (Colorscheme: solarized)
sudo add-apt-repository ppa:gnome-terminator

sudo apt-get update

sudo apt-get install terminator

For solarized colorscheme, update terminator config according to screenshot2.png


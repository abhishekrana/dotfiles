

wget https://github.com/neovim/neovim/releases/download/v0.3.1/nvim.appimage

true-color:
tmux - starting from version 2.2 (support since 427b820...)
sudo apt install xclip

pip install pycscope # for devjoe/vim-codequery
wget https://netcologne.dl.sourceforge.net/project/codequery/Release_version_0.21.0/Linux_64bit/codequery-qt5-0.21.0-Linux-amd64.deb
dpkg

git clone https://github.com/ruben2020/codequery

cd codequery
mkdir build
cd build
#Step 5: Run cmake (first line for Qt5 OR second line for Qt4 OR third line for no GUI).
cmake -G "Unix Makefiles" -DBUILD_QT5=ON ..
cmake -G "Unix Makefiles" -DBUILD_QT5=OFF ..
cmake -G "Unix Makefiles" -DNO_GUI=ON ..
# change install path
cmake -DCMAKE_INSTALL_PREFIX="/home/johndoe/tools/" -G "Unix Makefiles" -DBUILD_QT5=ON ..
# automatically adds bin in output path
cmake -DCMAKE_INSTALL_PREFIX="~/" -G "Unix Makefiles" -DBUILD_QT5=ON .
make
make install

pip3 install greenlet
pip3 install --user neovim
pip3 install --user --upgrade neovim
pip3 install jedi
pip3 install --upgrade jedi
mkdir -p ~/.local/share/fonts
cd ~/.local/share/fonts && wget https://github.com/ryanoasis/nerd-fonts/releases/download/v2.0.0/DejaVuSansMono.zip && unzip DejaVuSansMono.zip && rm DejaVuSansMono.zip
# cd ~/.local/share/fonts && wget https://github.com/ryanoasis/nerd-fonts/releases/download/v2.0.0/DroidSansMono.zip && unzip DroidSansMono.zip && rm DroidSansMono.zip



let s:uname = system("uname")
if s:uname == "Darwin\n"
    set runtimepath+=/Users/ox/.random/repos/github.com/Shougo/dein.vim
    call dein#begin('/Users/ox/.random')
    let g:python3_host_prog = '/usr/local/bin/python3'
else
    set runtimepath+=/home/ox/.random/repos/github.com/Shougo/dein.vim
    call dein#begin('/home/ox/.random')
    let g:python3_host_prog = '/usr/bin/python3'
endif



Ctrl+t
Ctrl+r

Ctrl+t .css$


git clone --depth 1 https://github.com/junegunn/fzf.git ~/.fzf
~/.fzf/install
# Update Plugin
#cd ~/.fzf && git pull && ./install

# CTRL-T - Paste the selected files and directories onto the command line
# CTRL-R - Paste the selected command from history onto the command line


:Gbrowse
:Gblame


curl https://beyondgrep.com/ack-2.24-single-file > ~/bin/ack && chmod 0755 ~/bin/ack

apt download silversearcher-ag
dpkg -x silversearcher-ag_0.31.0-2_amd64.deb ./

xset r rate 200  30

PlugUpdate
PlugUpgrade


deoplete
:UpdateRemotePlugins

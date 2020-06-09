TMUX_VERSION=3.1a
ROOT_PATH="$HOME/aSk"

sudo apt-get update

## Install Tmux
git clone https://github.com/tmux-plugins/tpm ~/.tmux/plugins/tpm
wget https://raw.githubusercontent.com/abhishekrana/dotfiles/master/tmux/tmux.conf -O ~/aSk/tmux.conf
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

# Manually run (inside tmux)
# <leader_key> I


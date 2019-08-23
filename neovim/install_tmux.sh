
### Install Tmux
# TMUX_VERSION=2.9
TMUX_MAJOR_VERSION=3.0
TMUX_MINOR_VERSION=-rc4
# git clone https://github.com/tmux-plugins/tpm ~/.tmux/plugins/tpm
# wget https://raw.githubusercontent.com/abhishekrana/dotfiles/master/tmux/.tmux.conf -O ~/.tmux.conf
# <leader_key> I
# if [ "$EUID" -eq 0 ];then
	echo "Install Tmux ..."
	# Root user
	apt-get -y install sudo
	sudo apt-get -y install wget tar libevent-dev libncurses-dev

	cd /tmp
	wget https://github.com/tmux/tmux/releases/download/${TMUX_MAJOR_VERSION}/tmux-${TMUX_MAJOR_VERSION}${TMUX_MINOR_VERSION}.tar.gz
	tar xf tmux-${TMUX_MAJOR_VERSION}${TMUX_MINOR_VERSION}.tar.gz
	rm -f tmux-${TMUX_MAJOR_VERSION}${TMUX_MINOR_VERSION}.tar.gz
	cd tmux-${TMUX_MAJOR_VERSION}${TMUX_MINOR_VERSION}
	./configure
	make
	sudo make install
	cd -
	sudo rm -rf /usr/local/src/tmux-\*
	# sudo mv tmux-${TMUX_MAJOR_VERSION}${TMUX_MINOR_VERSION} /usr/local/src
	sudo mv tmux-${TMUX_MAJOR_VERSION}${TMUX_MINOR_VERSION}/tmux ~/aSk/bin/
	echo "Installed Tmux"
# fi


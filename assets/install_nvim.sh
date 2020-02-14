# NEOVIM_VERSION="nightly"
NEOVIM_VERSION="v0.4.3"
CURRENT_VERSION=$(nvim --version | head -1 | cut -d ' ' -f 2)

install_neovim()
{
	pip3 install --upgrade neovim
	pip3 install --user --upgrade neovim	# Install without no root access
	curl -LO https://github.com/neovim/neovim/releases/download/${NEOVIM_VERSION}/nvim.appimage
	chmod u+x nvim.appimage
	./nvim.appimage --appimage-extract
	cp -r squashfs-root/usr/* ~/aSk/
	rm -rf squashfs-root
	rm nvim.appimage
	cp ~/aSk/bin/nvim ~/aSk/bin/vim
}

echo $CURRENT_VERSION
if ! [ -x "$(command -v nvim)" ];then
	install_neovim
	exit 0
fi

if [[ $CURRENT_VERSION != $NEOVIM_VERSION ]];then
	install_neovim
else
	echo "nvim already installed"
fi




NEOVIM_VERSION="nightly"

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

# pip install --upgrade pip
# pip3 install --upgrade neovim
# pip3 install --user --upgrade neovim	# Install without no root access

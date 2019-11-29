ROOT_PATH="$HOME/aSk"
mkdir -p $ROOT_PATH

## Setup bashrc config
cp bash/.bashrc_aSk ~/aSk/
grep -q "bashrc_aSk" ~/.bashrc
if [[ $? != 0 ]];then
	echo '' >> ~/.bashrc
	echo '### aSk'  >> ~/.bashrc
	echo 'if [ -f ~/aSk/.bashrc_aSk ]; then
		source ~/aSk/.bashrc_aSk
	fi' >> ~/.bashrc
	echo "~/.bashrc updated"
else
	echo "~/.bashrc not updated"
fi



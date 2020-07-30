#!/bin/bash 

# rm -rf ~/.vscode/extensions

extensions=(
	alefragnani.project-manager
	almenon.arepl
	arcticicestudio.nord-visual-studio-code
	azemoh.one-monokai
	bbrakenhoff.solarized-light-custom
	braver.vscode-solarized
	christian-kohler.path-intellisense
	deerawan.vscode-dash
	donjayamanne.githistory
	eamodio.gitlens
	formulahendry.code-runner
	formulahendry.docker-explorer
	foxundermoon.shell-format
	gerane.theme-solarized-dark
	gerane.theme-solarized-light
	ginfuru.ginfuru-better-solarized-dark-theme
	gruntfuggly.todo-tree
	GitHub.vscode-pull-request-github
	humao.rest-client
	mhutchie.git-graph
	ms-azuretools.vscode-docker
	ms-python.python
	ms-python.vscode-pylance
	ms-vscode-remote.remote-containers
	ms-vscode-remote.remote-ssh
	ms-vscode-remote.remote-ssh-edit
	ms-vscode-remote.remote-wsl
	ms-vscode-remote.vscode-remote-extensionpack
	njpwerner.autodocstring
	randomfractalsinc.vscode-data-preview
	redhat.vscode-yaml
	ryanolsonx.solarized
	shd101wyy.markdown-preview-enhanced
	tyriar.sort-lines
	usernamehw.errorlens
	victormejia.one-monokai-darker
	visualstudioexptteam.vscodeintellicode
	vscodevim.vim
	vscode-snippet.snippet
	vscoss.vscode-ansible
	waderyan.gitblame
	wayou.vscode-todo-highlight
	wholroyd.jinja
	yzhang.markdown-all-in-one
	ziyasal.vscode-open-in-github
)

echo "Old extensions"
code --list-extensions

for extension in "${extensions[@]}"
do
	code --install-extension $extension

done

echo "Current extensions"
code --list-extensions

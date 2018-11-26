FROM ubuntu:16.04

LABEL maintainer "aSk"

ENV HOME /workspace
WORKDIR $HOME
RUN chmod -R a+w /workspace


### Install dependencies ###
RUN apt-get update && apt-get install -y \
	wget \
	git \
	cmake \
	build-essential \
	exuberant-ctags \
	sudo \
	locate \
	curl \
	unzip \
	tree \
	tmux \
	gdbserver \
	python-pip \
	python3-pip
# 	firefox
#	openssh-server
#	vim

RUN pip install --upgrade pip && \
	pip3 install --upgrade pip

RUN pip install \
	pudb


### aSk config ###
RUN cd /tmp && \
	wget https://raw.githubusercontent.com/abhishekrana/dotfiles/master/setup.sh && \
	chmod +x setup.sh && \
	./setup.sh


# ## SSH config. Username:root, Password:root
# RUN mkdir /var/run/sshd
# RUN echo 'root:root' | chpasswd
# RUN sed -i 's/#PermitRootLogin prohibit-password/PermitRootLogin yes/' /etc/ssh/sshd_config

# ## SSH login fix. Otherwise user is kicked off after login
# RUN sed 's@session\s*required\s*pam_loginuid.so@session optional pam_loginuid.so@g' -i /etc/pam.d/sshd


# 22	: SSH into the container.
# 7777	: Run gdbserver for remotely debugging.
# 9999	: Connect to the program from outside
EXPOSE 22 7777 9999
# CMD ["/usr/sbin/sshd", "-D"]


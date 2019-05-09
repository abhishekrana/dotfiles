#!/bin/bash

## xmodmap also called from /usr/lib/pm-utils/sleep.d/99ZZZ_resume_script_aSk.sh after System Resume

# Wait for the desktop to finish loading
sleep 5

# Remap keyboard keys
xmodmap ~/.Xmodmap

# xset r rate 200  30
xset r rate 300  30

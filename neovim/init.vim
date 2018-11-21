" Abhishek Rana
" https://github.com/abhishekrana

" Plugin Manager {{{
" Install vimplug if not installed
" let g:vim_bootstrap_langs = "c,python"
" let g:vim_bootstrap_editor = "nvim"				" nvim or vim

" let g:python_host_prog = '/usr/bin/python'
" let g:python3_host_prog = '/usr/bin/python3'

" let g:python_host_prog = '/home/abhishek/lib/anaconda3/bin/python'
" let g:python3_host_prog = '/home/abhishek/lib/anaconda3/bin/python3'

if empty(glob('~/.config/nvim/autoload/plug.vim'))
  silent !curl -fLo ~/.config/nvim/autoload/plug.vim --create-dirs
    \ https://raw.githubusercontent.com/junegunn/vim-plug/master/plug.vim
  autocmd VimEnter * PlugInstall --sync
endif

" }}}

" Plugins {{{
call plug#begin(expand('~/.config/nvim/plugged'))

" deoplete-jedi
" let g:python_host_prog = '/full/path/to/neovim2/bin/python'
" let g:python3_host_prog = '/full/path/to/neovim3/bin/python'

"" Productivity
Plug 'tpope/vim-surround'
Plug 'tpope/vim-commentary'
Plug 'tpope/vim-repeat'
Plug 'junegunn/fzf.vim'																				" Runs asynchronously
Plug 'mileszs/ack.vim'
Plug 'junegunn/fzf', { 'dir': '~/.fzf', 'do': './install --bin' }
Plug 'justinmk/vim-sneak'
Plug 'dbeniamine/cheat.sh-vim'
Plug 'terryma/vim-multiple-cursors'
" Plug 'KabbAmine/zeavim.vim'
" Plug 'jiangmiao/auto-pairs'																			" Insert or delete brackets, parens, quotes in pair

" Plug 'Shougo/unite.vim'
" Plug 'devjoe/vim-codequery'
Plug 'ludovicchabant/vim-gutentags'																	" Automatic Tag generation
" Plug 'skywind3000/gutentags_plus'

"" Snippets
Plug 'SirVer/ultisnips'
" Plug 'honza/vim-snippets'
" Plug 'Shougo/neosnippet.vim'
" Plug 'Shougo/neosnippet-snippets'

"" Code completion/linting
" Plug 'neomake/neomake'																			" ale Slows down nvim in C/C++ code
" Plug 'w0rp/ale'																						" Asynchronous linting/fixing for Vim and Language Server Protocol (LSP) integration


""" Debugging
" Plug 'sakhnik/nvim-gdb'
Plug 'huawenyu/neogdb.vim'


"" ---------- CODE COMPLETION 1 ----------

" " pip install --upgrade jedi 
" " Jedi - an awesome autocompletion/static analysis library for Python
" " Used in Vim (jedi-vim, YouCompleteMe, deoplete-jedi, completor.vim)
" Plug 'davidhalter/jedi'

" " Dark powered asynchronous completion framework for neovim/Vim8
" Plug 'Shougo/deoplete.nvim', { 'do': ':UpdateRemotePlugins'  }


" " jedi-vim (with code completion disabled) for documentation and code traversal in python
" " deoplete-jedi (asynchronous) for code completion

" " To cycle through code-completion suggestions
" Plug 'ervandew/supertab'
" " jedi-vim is a VIM binding to the autocompletion library Jedi. Libraries are cached. Supports virtual env
" " [Dependency: supertab, jedi]
" Plug 'davidhalter/jedi-vim'

" "" Python
" " Implements support for Python plugins in Neovim
" Plug 'neovim/python-client'
" " [Dependency: deoplete.nvim, jedi, python-client]
" Plug 'zchee/deoplete-jedi'

" "" C/C++
" " apt-get install clang
" " [Dependency: deoplete.nvim]
" Plug 'tweekmonster/deoplete-clang2'

" "" C/C++
" " Plug 'zchee/libclang-python3'
" " [Dependency: deoplete.nvim]
" " Plug 'zchee/deoplete-clang'


"" ---------- CODE COMPLETION 2 ----------
Plug 'Valloric/YouCompleteMe', { 'do': './install.py --clang-completer' } " Conflicts with jedi-vim


"" ---------- CODE COMPLETION 3 ----------
" Conflicts with jedi-vim
" Plug 'python-mode/python-mode', { 'branch': 'develop' }


" Git
Plug 'tpope/vim-fugitive'
Plug 'airblade/vim-gitgutter'
" Plug 'Xuyuanp/nerdtree-git-plugin'																" A plugin of NERDTree showing git status 


" Key conflict
" Plug 'christoomey/vim-tmux-navigator'																" Seamless navigation between tmux panes and vim splits

"" UI
Plug 'Shougo/echodoc.vim'																			"Print documents in echo area while auto-completion
Plug 'scrooloose/nerdtree'
Plug 'lifepillar/vim-solarized8'
Plug 'vim-airline/vim-airline'
Plug 'vim-airline/vim-airline-themes'
Plug 'edkolev/tmuxline.vim'
Plug 'altercation/vim-colors-solarized'
" Plug 'tiagofumo/vim-nerdtree-syntax-highlight'
Plug 'ryanoasis/vim-devicons'																		" Always load the vim-devicons as the very last one
" Plug 'sheerun/vim-polyglot'																			" A collection of language packs for Vim.

" Plug 'neomake/neomake'

Plug 'majutsushi/tagbar'
" Plug 'scrooloose/syntastic'

"" Vim-Session
Plug 'xolox/vim-misc'
Plug 'xolox/vim-session'																			" Requires vim-misc

"" Color

" Plug 'altercation/vim-colors-solarized'

" Plug 'benmills/vimux'
" Plug 'xolox/vim-misc'
" Plug 'xolox/vim-session'
" Plug 'justinmk/vim-sneak'

" Plug 'Valloric/YouCompleteMe'	" Conflicts with jedi-vim
" Plug 'davidhalter/jedi-vim'

" vim-codequery
" Plug 'Shougo/unite.vim'
" Plug 'devjoe/vim-codequery'
" Plug 'tpope/vim-dispatch'
" Plug 'mileszs/ack.vim'

" Plug 'ggreer/the_silver_searcher'

" Plug 'octol/vim-cpp-enhanced-highlight'
" Plug 'scrooloose/syntastic'




"*****************************************************************************
"" Custom bundles
"*****************************************************************************

" c
" Plug 'vim-scripts/c.vim', {'for': ['c', 'cpp']}
" Plug 'ludwig/split-manpage.vim'


" python
"" Python Bundle
" Plug 'davidhalter/jedi-vim'
" Plug 'raimon49/requirements.txt.vim', {'for': 'requirements'}


"*****************************************************************************
"*****************************************************************************

"" Include user's extra bundle
" if filereadable(expand("~/.config/nvim/local_bundles.vim"))
"   source ~/.config/nvim/local_bundles.vim
" endif







call plug#end()
" }}}

" System Settings {{{

" Remove junk chars if using Terminator (https://launchpad.net/terminator).
" Add in ~/.bashrc
" export VTE_VERSION="100"
set guicursor=
autocmd OptionSet guicursor noautocmd set guicursor=


" .vimrc
" export TERM=xterm-256color


" Neovim settings
" let $NVIM_TUI_ENABLE_TRUE_COLOR=1
set termguicolors

" set relativenumber number
set number

set encoding=UTF-8																					" ryanoasis/vim-devicons, YouCompleteMe


" Uncomment the following to have Vim jump to the last position when reopening a file
if has("autocmd")
  au BufReadPost * if line("'\"") > 1 && line("'\"") <= line("$") | exe "normal! g'\"" | endif
endif
autocmd BufRead * normal zz

" set textwidth=80
set formatoptions+=t

set backspace=indent,eol,start

" make Vim use 256 colors
" set t_Co=256
" TODO
" colorscheme solarized8_light
" colorscheme solarized8_flat

" incsearch: search as characters are entered
set ic

" highlight matches
set hlsearch 

" show line numbers
set nu

" number of visual spaces per TAB when a file is read by vim 
set tabstop=4

" size of an indent using << or >>
set shiftwidth=4 

" number of spaces in tab when editing
set softtabstop=4	


" visual autocomplete for command menu
set wildmenu

" highlight matching [{()}]}]
set showmatch

"Auto indent
set ai

"Smart indent
set si

" Enable syntax highlighting, text coloring based on syntax
" syntax enable

" Don't redraw while executing macros (good performance config)
set lazyredraw

" Set 5 lines to the cursor - when moving vertically using j/k
set so=5

" With both ignorecase and smartcase turned on, a search is case-insensitive if
" you enter the search string in ALL lower case else case-sensitive 
set smartcase

" Ignore case when searching
set ignorecase

" highlight current line
"set cursorline 

" tabs are spaces. Don't do for Makefile
"set expandtab 

"Yank across across terminal (Linux)
set clipboard=unnamedplus

" show relative line numbers
" set relativenumber 

set colorcolumn=100
" hi ColorColumn guibg=#2d2d2d ctermbg=246
hi ColorColumn guibg=#000000 ctermbg=254


" Dropdown menu color
"highlight Pmenu ctermfg=15 ctermbg=blue guifg=#ffffff guibg=#000080
highlight Pmenu ctermfg=15 ctermbg=67 guifg=#ffffff guibg=#000080
" NumberLine color
highlight LineNr ctermbg=254
" Highlighted word
highlight Search cterm=NONE ctermfg=white
" highlight Pmenu ctermbg=8 guibg=#606060
" highlight PmenuSel ctermbg=1 guifg=#dddd00 guibg=#1f82cd
" highlight PmenuSbar ctermbg=0 guibg=#d6d6d6

" 2 lines above/below cursor when scrolling
set scrolloff=2

" no redraws in macros
set lazyredraw


" }}}

" Mapping {{{
"""""""""""""""""""""""""""""""""""""""""""
" Mapping 
"""""""""""""""""""""""""""""""""""""""""""
map <space> <leader>

noremap <leader>f :Files<CR>
noremap <leader>w :w<CR>
noremap <leader>q :q<CR>
nnoremap <leader>F :call QuickfixToggle()<cr>
nnoremap <leader>i :call PasteToggle()<cr>
nnoremap <leader>= :call BackgroundToggle()<cr>
noremap <leader>[ :cprev<CR>
noremap <leader>] :cnext<CR>
"noremap <F10> ::!ctags -R --c++-kinds=+p --fields=+iaSl --extra=+q; find . -name "*.c" -o -name "*.cpp" -o -name "*.h" -o -name "*.hpp" > cscope.files; cscope -q -R -b -i cscope.files <CR>
noremap <S-l> <ESC>$
noremap <S-h> <ESC>^
noremap <S-j> <ESC>:call cursor(0, len(getline('.'))/2)<cr> 

noremap ff <Esc>:w<CR>
inoremap ff <Esc>:w<CR>
inoremap jk <Esc>:w<CR>
" nnoremap <leader><leader> <Esc>:w<CR>

" nnoremap <leader>h :prev<CR>
" nnoremap <leader>l :next<CR>
nnoremap <leader>h <C-w>h
nnoremap <leader>l <C-w>l
nnoremap <leader>j :bprev<CR>
nnoremap <leader>k :bnext<CR>
nnoremap <leader>y :set paste<CR>
nnoremap <leader>Y :set nopaste<CR>
nnoremap <leader>c ciw


" Session:
" It will not save the changes to any files that you've made
" vim -S ~/mysession.vim
set ssop-=options    " do not store global and local values in a session
set ssop-=folds      " do not store folds
nnoremap <leader>z :SaveSession! .session
nnoremap <leader>Z :OpenSession! .session<cr>

" word to printf('',word)
noremap <leader>p yiwoprint('', )<ESC><left><left><left><left>p<right><right><right>p<right>^<down>
noremap <leader>o yiwologging.debug(' {}'.format())<ESC>bbbb<right>pwwwp<right>^<down>
" noremap <leader>i yiwologging.debug('')<ESC><left><left>p<right>^<down>

" Edit Vimrc
nnoremap <leader>ev :vsplit $MYVIMRC<cr>
" Source Vimrc
nnoremap <leader>sv :source $MYVIMRC<cr>

"Easier moving of code blocks
vnoremap < <gv
vnoremap > >gv

nnoremap <C-i> <C-o>
nnoremap <C-o> <C-i>


nnoremap ; :
" map <esc> :noh<cr>


" }}}

" Folding {{{
"""""""""""""""""""""""""""""""""""""""""""
" Folding 
"""""""""""""""""""""""""""""""""""""""""""
augroup filetype_vim
	autocmd!
    autocmd FileType vim setlocal foldmethod=marker
augroup END

set modelines=1

" }}}


"*****************************************************************************
"" Visual Settings
"*****************************************************************************
syntax on
set ruler
set mousemodel=popup
" set guioptions=egmrti
" set gfn=Monospace\ 10

" let g:CSApprox_loaded = 1
" " IndentLine
" let g:indentLine_enabled = 1
" let g:indentLine_concealcursor = 0
" let g:indentLine_char = '┆'
" let g:indentLine_faster = 1



"" Disable the blinking cursor.
set gcr=a:blinkon0
set scrolloff=3

"" Status bar
set laststatus=2

"" Use modeline overrides
set modeline
set modelines=10

set title
set titleold="Terminal"
set titlestring=%F

set statusline=%F%m%r%h%w%=(%{&ff}/%Y)\ (line\ %l\/%L,\ col\ %c)\

" Search mappings: These will make it so that going to the next one in a
" search will center on the line it's found in.
nnoremap n nzzzv
nnoremap N Nzzzv

if exists("*fugitive#statusline")
  set statusline+=%{fugitive#statusline()}
endif

" vim-airline
" let g:airline_theme = 'powerlineish'
let g:airline_theme = 'solarized'
let g:airline#extensions#syntastic#enabled = 1
let g:airline#extensions#branch#enabled = 1
let g:airline#extensions#tabline#enabled = 1
let g:airline#extensions#tagbar#enabled = 1
let g:airline_skip_empty_sections = 1

" deoplete.vim
let g:deoplete#enable_at_startup = 1
" set completeopt+=noinsert " Enable auto select feature
" set completeopt-=preview " Disable preview window
" let g:deoplete#sources#clang#libclang_path = '/usr/lib/llvm-3.8/lib/'

" deoplete-jedi
let g:deoplete#sources#jedi#show_docstring = 1
" Set the Python interpreter path to use for the completion server. It defaults to "python", i.e. 
" the first available python in $PATH. Note: This is different from Neovim's Python (:python) in general.
" let g:deoplete#sources#jedi#python_path = '/home/abhishek/lib/anaconda3/bin/python' 

" jedi-vim
let g:jedi#completions_enabled = 0
let g:jedi#popup_on_dot = 1
let g:jedi#goto_assignments_command = "<leader>g"
let g:jedi#goto_definitions_command = "<leader>d"
let g:jedi#documentation_command = "K"
" let g:jedi#usages_command = "<leader>n"
" let g:jedi#rename_command = "<leader>r"
" let g:jedi#completions_command = "<C-Space>"
" autocmd BufWinEnter '__doc__' setlocal bufhidden=delete
let g:jedi#smart_auto_mappings = 0 " Don't add 'import' automatically
set splitbelow	" The documentation is opened in a horizontally split buffer below
" let g:jedi#max_doc_height =
" let g:jedi#auto_close_doc = 0
let g:jedi#show_call_signatures = 1
" let g:jedi#show_call_signatures = 2
" autocmd CompleteDone * silent!
" autocmd InsertLeave,CompleteDone * if pumvisible() == 0 | echo 'aa' | endif
" Use <TAB> to select the popup menu:
" inoremap <expr> <Tab> pumvisible() ? "\<C-n>" : "\<Tab>"
" inoremap <expr> <S-Tab> pumvisible() ? "\<C-p>" : "\<S-Tab>"


"" zeavim docset
" nmap <leader>x <Plug>Zeavim
" vmap <leader>x <Plug>ZVVisSelection
" nmap gx <Plug>ZVOperator
" nmap <leader><leader>x <Plug>ZVKeyDocset

" supertab
let g:SuperTabDefaultCompletionType = "<c-n>"

" tmuxline
let g:tmuxline_powerline_separators = 0
" If you set airline theme manually, make sure the airline-tmuxline extension is disabled, so it doesn't overwrite the theme
let g:airline#extensions#tmuxline#enabled = 0
" let g:tmuxline_separators = {
"     \ 'left' : '',
"     \ 'left_alt': '>',
"     \ 'right' : '',
"     \ 'right_alt' : '<',
"     \ 'space' : ' '}

" codequery
" let g:codequery_trigger_build_db_when_db_not_found = 1
" 0 => search from project dir (git root directory -> then the directory containing xxx.db file)
" 1 => search from the directory containing current file"
" let g:codequery_find_text_from_current_file_dir = 0

" let g:codequery_auto_switch_to_find_text_for_wrong_filetype = 1
" set tags+=./python_tags;/

" echodoc
set noshowmode
let g:echodoc_enable_at_startup = 1
" let g:enable_force_overwrite = 1

"" YouCompleteMe
" When this option is set to '1', YCM will add the 'preview' string to Vim's
" 'completeopt' option. 
" :set completeopt?
" When 'preview' is present in 'completeopt', YCM will use the 'preview' window
" at the top of the file to store detailed information about the current
" completion candidate (but only if the candidate came from the semantic engine).
" For instance, it would show the full function prototype and all the function
" overloads in the window if the current completion is a function name.
" let g:ycm_add_preview_to_completeopt = 1
" let g:ycm_autoclose_preview_window_after_completion = 1
" This option is irrelevant if |g:ycm_autoclose_preview_window_after_completion| 
" is set or if no 'preview' window is triggered.
" let g:ycm_autoclose_preview_window_after_insertion = 1
let g:ycm_complete_in_comments = 1
" The only supported tag format is the Exuberant Ctags format [72]. The format
" from "plain" ctags is NOT supported. Ctags needs to be called with the '--
" fields=+l' option (that's a lowercase 'L', not a one) because YCM needs the
" 'language:<lang>' field in the tags output.
" This option is off by default because it makes Vim slower if your tags are on a
" let g:ycm_collect_identifiers_from_tags_files = 1
" let g:ycm_server_python_interpreter = '/home/abhishek/lib/anaconda3/bin/python'
" let g:ycm_global_ycm_extra_conf = ''

" vim-multi-cursors
let g:multi_cursor_use_default_mapping = 0
let g:multi_cursor_start_word_key      = '<C-n>'
let g:multi_cursor_select_all_word_key = '<A-n>'
let g:multi_cursor_start_key           = 'g<C-n>'
let g:multi_cursor_select_all_key      = 'g<A-n>'
let g:multi_cursor_next_key            = '<C-n>'
let g:multi_cursor_prev_key            = '<C-p>'
let g:multi_cursor_skip_key            = '<C-x>'
let g:multi_cursor_quit_key            = '<Esc>'


"*****************************************************************************
"" Abbreviations
"*****************************************************************************
"" NERDTree configuration
" let g:NERDTreeChDirMode=2
" let g:NERDTreeIgnore=['\.rbc$', '\~$', '\.pyc$', '\.db$', '\.sqlite$', '__pycache__']
" let g:NERDTreeSortOrder=['^__\.py$', '\/$', '*', '\.swp$', '\.bak$', '\~$']
" let g:NERDTreeShowBookmarks=1
" let g:nerdtree_tabs_focus_on_files=1
" let g:NERDTreeMapOpenInTabSilent = '<RightMouse>'
" let g:NERDTreeWinSize = 50
" set wildignore+=*/tmp/*,*.so,*.swp,*.zip,*.pyc,*.db,*.sqlite
" nnoremap <silent> <F2> :NERDTreeFind<CR>
" nnoremap <silent> <F3> :NERDTreeToggle<CR>

" grep.vim
" nnoremap <silent> <leader>f :Rgrep<CR>
" let Grep_Default_Options = '-IR'
" let Grep_Skip_Files = '*.log *.db'
" let Grep_Skip_Dirs = '.git node_modules'

" vimshell.vim
" let g:vimshell_user_prompt = 'fnamemodify(getcwd(), ":~")'
" let g:vimshell_prompt =  '$ '

" terminal emulation
" if g:vim_bootstrap_editor == 'nvim'
"   nnoremap <silent> <leader>sh :terminal<CR>
" else
"   nnoremap <silent> <leader>sh :VimShellCreate<CR>
" endif

"*****************************************************************************
"" Functions
"*****************************************************************************
if !exists('*s:setupWrapping')
  function s:setupWrapping()
    set wrap
    set wm=2
    set textwidth=79
  endfunction
endif

let g:quickfix_is_open = 0
function! QuickfixToggle()
	if g:quickfix_is_open
		cclose
		let g:quickfix_is_open = 0
		execute g:quickfix_return_to_window . "wincmd w"
	else
		let g:quickfix_return_to_window = winnr()
		copen
		let g:quickfix_is_open = 1
	endif
endfunction

" PASTE:
let g:paste_flag = 0
function! PasteToggle()
    if g:paste_flag
        let g:paste_flag = 0
		set nopaste
		inoremap ff <Esc>:w<CR>
    else
        let g:paste_flag = 1
		set paste
		inoremap ff
    endif
endfunction


let g:background_flag = 0
function! BackgroundToggle()
	if g:background_flag
		set background=light
		let g:background_flag = 0
	else
		set background=dark
		let g:background_flag = 1
	endif
endfunction

"*****************************************************************************
"" Autocmd Rules
"*****************************************************************************
"" The PC is fast enough, do syntax highlight syncing from start unless 200 lines
augroup vimrc-sync-fromstart
  autocmd!
  autocmd BufEnter * :syntax sync maxlines=200
augroup END

"" Remember cursor position
augroup vimrc-remember-cursor-position
  autocmd!
  autocmd BufReadPost * if line("'\"") > 1 && line("'\"") <= line("$") | exe "normal! g`\"" | endif
augroup END

"" txt
augroup vimrc-wrapping
  autocmd!
  autocmd BufRead,BufNewFile *.txt call s:setupWrapping()
augroup END

"" make/cmake
augroup vimrc-make-cmake
  autocmd!
  autocmd FileType make setlocal noexpandtab
  autocmd BufNewFile,BufRead CMakeLists.txt setlocal filetype=cmake
augroup END

"" Start NerdTree automatically when vim starts
autocmd VimEnter * NERDTree | wincmd p

"" Close vim if the only window left open is a NERDTree
autocmd BufEnter * if (winnr("$") == 1 && exists("b:NERDTree") && b:NERDTree.isTabTree()) | q | endif


set autoread

"*****************************************************************************
"" Mappings
"*****************************************************************************

" session management
" nnoremap <leader>so :OpenSession<Space>
" nnoremap <leader>ss :SaveSession<Space>
" nnoremap <leader>sd :DeleteSession<CR>
" nnoremap <leader>sc :CloseSession<CR>

"" Opens an edit command with the path of the currently edited file filled in
" noremap <Leader>e :e <C-R>=expand("%:p:h") . "/" <CR>

"" Opens a tab edit command with the path of the currently edited file filled
" noremap <Leader>te :tabe <C-R>=expand("%:p:h") . "/" <CR>

"" fzf.vim
" set wildmode=list:longest,list:full
" set wildignore+=*.o,*.obj,.git,*.rbc,*.pyc,__pycache__
" let $FZF_DEFAULT_COMMAND =  "find * -path '*/\.*' -prune -o -path 'node_modules/**' -prune -o -path 'target/**' -prune -o -path 'dist/**' -prune -o  -type f -print -o -type l -print 2> /dev/null"

" The Silver Searcher
" if executable('ag')
"   let $FZF_DEFAULT_COMMAND = 'ag --hidden --ignore .git -g ""'
"   set grepprg=ag\ --nogroup\ --nocolor
" endif

" ripgrep
" if executable('rg')
"   let $FZF_DEFAULT_COMMAND = 'rg --files --hidden --follow --glob "!.git/*"'
"   set grepprg=rg\ --vimgrep
"   command! -bang -nargs=* Find call fzf#vim#grep('rg --column --line-number --no-heading --fixed-strings --ignore-case --hidden --follow --glob "!.git/*" --color "always" '.shellescape(<q-args>).'| tr -d "\017"', 1, <bang>0)
" endif

" cnoremap <C-P> <C-R>=expand("%:p:h") . "/" <CR>
" nnoremap <silent> <leader>b :Buffers<CR>
" nnoremap <silent> <leader>e :FZF -m<CR>

" snippets
let g:UltiSnipsExpandTrigger="<tab>"
let g:UltiSnipsJumpForwardTrigger="<tab>"
let g:UltiSnipsJumpBackwardTrigger="<c-b>"
let g:UltiSnipsEditSplit="vertical"

" syntastic
let g:syntastic_always_populate_loc_list=1
let g:syntastic_error_symbol='✗'
let g:syntastic_warning_symbol='⚠'
let g:syntastic_style_error_symbol = '✗'
let g:syntastic_style_warning_symbol = '⚠'
let g:syntastic_auto_loc_list=1
let g:syntastic_aggregate_errors = 1

" Tagbar
nmap <silent> <F4> :TagbarToggle<CR>
let g:tagbar_sort = 0
let g:tagbar_compact = 1
" let g:tagbar_autofocus = 1
autocmd VimEnter * nested :call tagbar#autoopen(1)
" autocmd FileType c,cpp,py nested :TagbarOpen

" Disable visualbell
set noerrorbells visualbell t_vb=
if has('autocmd')
  autocmd GUIEnter * set visualbell t_vb=
endif

"" Copy/Paste/Cut
if has('unnamedplus')
  set clipboard=unnamed,unnamedplus
endif


"" Open current line on GitHub
" nnoremap <Leader>o :.Gbrowse<CR>

"*****************************************************************************
"" Custom configs
"*****************************************************************************

" c
autocmd FileType c setlocal tabstop=4 shiftwidth=4 expandtab
autocmd FileType cpp setlocal tabstop=4 shiftwidth=4 expandtab


" python
" vim-python
augroup vimrc-python
  autocmd!
  autocmd FileType python setlocal expandtab shiftwidth=4 tabstop=8 colorcolumn=79
      \ formatoptions+=croq softtabstop=4
      \ cinwords=if,elif,else,for,while,try,except,finally,def,class,with
augroup END


" syntastic
let g:syntastic_python_checkers=['python', 'flake8']

" vim-airline
let g:airline#extensions#virtualenv#enabled = 1

" Syntax highlight
" Default highlight is better than polyglot
let g:polyglot_disabled = ['python']
let python_highlight_all = 1


" vim-session
let g:session_autosave = 'no'
let g:session_directory='.'

" sheerun/vim-polyglot
let g:python_highlight_all = 1

""" NERDTree
" function! NERDTreeHighlightFile(extension, fg, bg, guifg, guibg)
"  exec 'autocmd filetype nerdtree highlight ' . a:extension .' ctermbg='. a:bg .' ctermfg='. a:fg .' guibg='. a:guibg .' guifg='. a:guifg
"  exec 'autocmd filetype nerdtree syn match ' . a:extension .' #^\s\+.*'. a:extension .'$#'
" endfunction

" call NERDTreeHighlightFile('jade', 'green', 'none', 'green', '#151515')
" call NERDTreeHighlightFile('ini', 'yellow', 'none', 'yellow', '#151515')
" call NERDTreeHighlightFile('md', 'blue', 'none', '#3366FF', '#151515')
" call NERDTreeHighlightFile('yml', 'yellow', 'none', 'yellow', '#151515')
" call NERDTreeHighlightFile('config', 'yellow', 'none', 'yellow', '#151515')
" call NERDTreeHighlightFile('conf', 'yellow', 'none', 'yellow', '#151515')
" call NERDTreeHighlightFile('json', 'yellow', 'none', 'yellow', '#151515')
" call NERDTreeHighlightFile('html', 'yellow', 'none', 'yellow', '#151515')
" call NERDTreeHighlightFile('styl', 'cyan', 'none', 'cyan', '#151515')
" call NERDTreeHighlightFile('css', 'cyan', 'none', 'cyan', '#151515')
" call NERDTreeHighlightFile('coffee', 'Red', 'none', 'red', '#151515')
" call NERDTreeHighlightFile('js', 'Red', 'none', '#ffa500', '#151515')
" call NERDTreeHighlightFile('php', 'Magenta', 'none', '#ff00ff', '#151515')


""" vim-sneak
let g:sneak#s_next = 1

""" neomake
" " When writing a buffer (no delay).
" call neomake#configure#automake('w')
" " When writing a buffer (no delay), and on normal mode changes (after 750ms).
" call neomake#configure#automake('nw', 750)
" " When reading a buffer (after 1s), and when writing (no delay).
" call neomake#configure#automake('rw', 1000)
" " Full config: when writing or reading a buffer, and on changes in insert and
" " normal mode (after 1s; no delay when writing).
" call neomake#configure#automake('nrwi', 500)

"*****************************************************************************
"*****************************************************************************

"" Include user's local vim config
" if filereadable(expand("~/.config/nvim/local_init.vim"))
"   source ~/.config/nvim/local_init.vim
" endif

"*****************************************************************************
"" Convenience variables
"*****************************************************************************

" vim-airline
if !exists('g:airline_symbols')
  let g:airline_symbols = {}
endif

if !exists('g:airline_powerline_fonts')
  let g:airline#extensions#tabline#left_sep = ' '
  let g:airline#extensions#tabline#left_alt_sep = '|'
  let g:airline_left_sep          = '▶'
  let g:airline_left_alt_sep      = '»'
  let g:airline_right_sep         = '◀'
  let g:airline_right_alt_sep     = '«'
  let g:airline#extensions#branch#prefix     = '⤴' "➔, ➥, ⎇
  let g:airline#extensions#readonly#symbol   = '⊘'
  let g:airline#extensions#linecolumn#prefix = '¶'
  let g:airline#extensions#paste#symbol      = 'ρ'
  let g:airline_symbols.linenr    = '␊'
  let g:airline_symbols.branch    = '⎇'
  let g:airline_symbols.paste     = 'ρ'
  let g:airline_symbols.paste     = 'Þ'
  let g:airline_symbols.paste     = '∥'
  let g:airline_symbols.whitespace = 'Ξ'
else
  let g:airline#extensions#tabline#left_sep = ''
  let g:airline#extensions#tabline#left_alt_sep = ''

  " powerline symbols
  let g:airline_left_sep = ''
  let g:airline_left_alt_sep = ''
  let g:airline_right_sep = ''
  let g:airline_right_alt_sep = ''
  let g:airline_symbols.branch = ''
  let g:airline_symbols.readonly = ''
  let g:airline_symbols.linenr = ''
endif


let g:airline#extensions#tabline#buffer_idx_mode = 1

nmap <leader>1 <Plug>AirlineSelectTab1
nmap <leader>2 <Plug>AirlineSelectTab2
nmap <leader>3 <Plug>AirlineSelectTab3
nmap <leader>4 <Plug>AirlineSelectTab4
nmap <leader>5 <Plug>AirlineSelectTab5
nmap <leader>6 <Plug>AirlineSelectTab6
nmap <leader>7 <Plug>AirlineSelectTab7
nmap <leader>8 <Plug>AirlineSelectTab8
nmap <leader>9 <Plug>AirlineSelectTab9


" :CheckHealth

"" vim-surround
" cs"'

" autocmd CompleteDone * pclose " To close preview window of deoplete automatically


" set background=dark
if !empty(glob('~/.config/nvim/plugged/vim-solarized8/colors/solarized8_flat.vim'))
	colorscheme solarized8_flat
endif

" YouCompleteMe
" cd ~/.config/nvim/plugged/YouCompleteMe
" python3 install.py --clang-completer
" https://github.com/neovim/neovim/tree/master/contrib/YouCompleteMe

" python -c "import jedi, sys; print(jedi.Script('import tensorflow',sys_path=sys.path).completions())" 


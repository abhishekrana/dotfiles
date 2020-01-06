" Abhishek Rana
" https://github.com/abhishekrana

" Plugin Manager {{{

if empty(glob('~/.config/nvim/autoload/plug.vim'))
  silent !curl -fLo ~/.config/nvim/autoload/plug.vim --create-dirs
    \ https://raw.githubusercontent.com/junegunn/vim-plug/master/plug.vim
  autocmd VimEnter * PlugInstall --sync
endif

" }}}

" Plugins {{{

call plug#begin(expand('~/.config/nvim/plugged'))

" Intellisense code engine, auto-completion, linting, code fixing
Plug 'neoclide/coc.nvim', {'branch': 'release'}
let g:coc_global_extensions = [
  \ 'coc-json',
  \ 'coc-python',
  \ 'coc-snippets',
  \ 'coc-pairs',
  \ 'coc-explorer',
  \ 'coc-yank',
  \ 'coc-highlight',
  \ 'coc-git',
  \ 'coc-markdownlint',
  \ 'coc-lists',
  \ 'coc-marketplace',
  \ 'coc-yaml',
  \ 'coc-terminal',
  \ 'coc-git',
  \ 'coc-tag',
  \ ]

Plug 'honza/vim-snippets'
" Plug 'vim-vdebug/vdebug'
" Plug 'sakhnik/nvim-gdb', { 'do': ':!./install.sh \| UpdateRemotePlugins' }

" Fuzzy file finding, project searching, file browsing
Plug 'junegunn/fzf', { 'dir': '~/.fzf', 'do': './install --all' }
Plug 'junegunn/fzf.vim'
Plug 'scrooloose/nerdtree'
"Plug 'tsony-tsonev/nerdtree-git-plugin'
Plug 'Xuyuanp/nerdtree-git-plugin'
Plug 'tiagofumo/vim-nerdtree-syntax-highlight'
" Plug 'jeetsukumaran/vim-buffergator'
Plug 'jlanzarotta/bufexplorer'

" Git
Plug 'tpope/vim-fugitive'
Plug 'tpope/vim-rhubarb'
" Plug 'airblade/vim-gitgutter'

" UI
Plug 'vim-airline/vim-airline'
Plug 'vim-airline/vim-airline-themes'
Plug 'edkolev/tmuxline.vim'
Plug 'lifepillar/vim-solarized8'
Plug 'altercation/vim-colors-solarized'
Plug 'ryanoasis/vim-devicons'
Plug 'mhinz/vim-startify'
Plug 'tmux-plugins/vim-tmux-focus-events'

" Vim-Session
" Plug 'xolox/vim-misc'
" Plug 'xolox/vim-session'

" Misc
" Plug 'machakann/vim-highlightedyank'
Plug 'tpope/vim-commentary'
Plug 'liuchengxu/vista.vim'
Plug 'majutsushi/tagbar'
" Plug 'jpalardy/vim-slime'
Plug 'fcpg/vim-osc52'
" Plug 'terryma/vim-multiple-cursors'
Plug 'JamshedVesuna/vim-markdown-preview'

" Plug 'unblevable/quick-scope' " hangs a lot everything
" Plug 'christoomey/vim-tmux-navigator'

call plug#end()

" }}}

" System Settings {{{

syntax enable							" enable syntax highlighting, text coloring based on syntax
set termguicolors						" set true colors
set number								" set relativenumber number
set encoding=UTF-8						" encoding
set backspace=indent,eol,start
set ic									" incsearch: search as characters are entered
set hlsearch							" highlight matches
set nu									" set relativenumber number
set tabstop=4							" number of visual spaces per TAB
set shiftwidth=4						" size of an indent using << or >>
set softtabstop=4						" number of spaces in tab when editing
set wildmenu							" visual autocomplete for command menu
set showmatch							" highlight matching braces
set ai									" auto indent
set si									" smart indent
set lazyredraw							" no redraws in macros
set smartcase							" With both ignorecase and smartcase turned on, a search is case-insensitive if
										" you enter the search string in ALL lower case else case-sensitive 
set ignorecase							" ignore case when searching
set cursorline							" highlight current line
set showcmd								" show (partial) command in the last line of the screen
set clipboard=unnamedplus				" yank across across terminal (Linux)
set relativenumber						" show relative line numbers
set formatoptions+=t					" auto-wrap text using textwidth value
" set textwidth=100
set scrolloff=4							" lines visible above/below cursor when scrolling vertically
set ruler
set laststatus=2						" status bar
set wildoptions=pum
set pumblend=10							" popup-menu transparency
set autowrite							" Save file when moving buffers
set autoread							" Read latest file when moving buffers
set statusline=%F%m%r%h%w%=(%{&ff}/%Y)\ (line\ %l\/%L,\ col\ %c)\
set fillchars+=vert:\					" Remove char in the vertical separation line


" set mousemodel=popup
" set gcr=a:blinkon0					" disable the blinking cursor.
" set title
" set titleold="Terminal"
" set titlestring=%F
" set colorcolumn=100
" hi VertSplit ctermbg=NONE guibg=NONE

" }}}

" Mapping {{{
"""""""""""""""""""""""""""""""""""""""""""
" Mapping 
"""""""""""""""""""""""""""""""""""""""""""
map <space> <leader>

" neoclide/coc.nvim - Most commands support CTRL-T / CTRL-X / CTRL-V key bindings
" to open in a new tab, a new split, or in a new vertical split
noremap <leader>f :GFiles<CR>
noremap <leader>F :Files<CR>
nmap <leader>t :BTags<CR>
nmap <leader>T :Tags<CR>
nnoremap <silent><leader>s :Rg <C-R>=expand("<cword>")<CR><CR>
nnoremap <silent><leader>S :Ag <C-R>=expand("<cword>")<CR><CR>
nmap <leader>R :Vista finder coc <C-R>
noremap <leader>v :call CocAction('format') \| w<CR>
noremap <leader>b :TagbarToggle<CR>
noremap <silent>c :ToggleBufExplorer<CR>


" Custom Mappings
noremap ff <Esc>:w<CR>
inoremap ff <Esc>:w<CR>
noremap <S-l> <ESC>$
noremap <S-h> <ESC>^
" noremap <S-j> <ESC>:call cursor(0, len(getline('.'))/2)<cr>
nnoremap <C-i> <C-o>
nnoremap <C-o> <C-i>
nnoremap ; :
nnoremap j gj
nnoremap k gk
vnoremap < <gv							" Easier moving of code blocks
vnoremap > >gv							" Easier moving of code blocks
nnoremap n nzzzv						" Going to the next one in a search will center on the line it's found in.
nnoremap N Nzzzv						" Going to the next one in a search will center on the line it's found in.
nnoremap * :keepjumps normal! mi*`i<CR>	" Dont jump to next word

" Custom <leader> Mappings
" noremap <leader>w :windo bd<CR>
" noremap <leader>w :bd<CR>
noremap <leader>Q :wqa<CR>
noremap <leader>q :bd<CR>
" noremap <leader>z :TagbarClose<CR> \| :mks!<CR> \| :TagbarOpen<CR>
noremap <leader>z :mks!<CR>
" noremap <leader>q :SaveSession! .session<CR>:qa<CR>
nnoremap <leader>h <C-w>h
nnoremap <leader>l <C-w>l
nnoremap <leader>j :bprev<CR>
nnoremap <leader>k :bnext<CR>
nnoremap <leader>n :NERDTreeToggle<CR>
nnoremap <leader>c ciw
noremap <leader>p yiwoprint('', )<ESC><left><left><left><left>p<right><right><right>p<right>^<down>
noremap <leader>o yiwologging.debug(' {}'.format())<ESC>bbbb<right>pwwwp<right>^<down>

" Session - It will not save the changes to any files that you've made
set ssop-=options    " do not store global and local values in a session
set ssop-=folds      " do not store folds
" nnoremap <leader>z :OpenSession! .session<cr>
" autocmd! VimEnter * OpenSession! .session<cr>

" Edit Vimrc
nnoremap <leader>ev :vsplit $MYVIMRC<cr>

"nnoremap <leader>F :call QuickfixToggle()<cr>
"nnoremap <leader>i :call PasteToggle()<cr>
"nnoremap <leader>= :call BackgroundToggle()<cr>
"noremap <leader>| :!ctags -R --c++-kinds=+p --fields=+iaSl --extra=+q; find . -name "*.c" -o -name "*.cpp" -o -name "*.h" -o -name "*.hpp" > cscope.files; cscope -q -R -b -i cscope.files <CR>
"noremap <leader>\ :!ctags -R --fields=+iaSl --extra=+q<CR>

noremap <F4> :!ctags -RV --python-kinds=-v --language-force=python --fields=+ailS --exclude=tmp --exclude=output --exclude=datasets <CR>
" --python-kinds=-v: supress tag entries for variables
" a	Access (or export) of class members
" f	File-restricted scoping [enabled]
" i	Inheritance information
" k	Kind of tag as a single letter [enabled]
" K	Kind of tag as full name
" l	Language of source file containing tag
" m	Implementation information
" n	Line number of tag definition
" s	Scope of tag definition [enabled]
" S	Signature of routine (e.g. prototype or parameter list)
" z	Include the "kind:" key in kind field	
" t	Type and name of a variable or typedef as "typeref:" field [enabled]


" }}}

" Plugins Config {{{

" === vim-session ===
let g:session_autosave = 'no'
let g:session_directory='.'

" === coc ===
" if hidden is not set, TextEdit might fail.
set hidden

" Some servers have issues with backup files, see #649
set nobackup
set nowritebackup

" Better display for messages
set cmdheight=2

" You will have bad experience for diagnostic messages when it's default 4000.
set updatetime=300

" don't give |ins-completion-menu| messages.
set shortmess+=c

" always show signcolumns
set signcolumn=yes

" Use tab for trigger completion with characters ahead and navigate.
" Use command ':verbose imap <tab>' to make sure tab is not mapped by other plugin.
inoremap <silent><expr> <TAB>
      \ pumvisible() ? "\<C-n>" :
      \ <SID>check_back_space() ? "\<TAB>" :
      \ coc#refresh()
inoremap <expr><S-TAB> pumvisible() ? "\<C-p>" : "\<C-h>"

function! s:check_back_space() abort
  let col = col('.') - 1
  return !col || getline('.')[col - 1]  =~# '\s'
endfunction

" Use <c-space> to trigger completion.
inoremap <silent><expr> <c-space> coc#refresh()

" Use <cr> to confirm completion, `<C-g>u` means break undo chain at current position.
" Coc only does snippet and additional edit on confirm.
inoremap <expr> <cr> pumvisible() ? "\<C-y>" : "\<C-g>u\<CR>"
" Or use `complete_info` if your vim support it, like:
" inoremap <expr> <cr> complete_info()["selected"] != "-1" ? "\<C-y>" : "\<C-g>u\<CR>"

" Use `[g` and `]g` to navigate diagnostics
nmap <silent> [g <Plug>(coc-diagnostic-prev)
nmap <silent> ]g <Plug>(coc-diagnostic-next)

" Remap keys for gotos
nmap <silent> <leader>d <Plug>(coc-definition)
nmap <silent> <leader>y <Plug>(coc-type-definition)
nmap <silent> <leader>i <Plug>(coc-implementation)
nmap <silent> <leader>r <Plug>(coc-references)

" Use K to show documentation in preview window
nnoremap <silent> K :call <SID>show_documentation()<CR>

function! s:show_documentation()
  if (index(['vim','help'], &filetype) >= 0)
    execute 'h '.expand('<cword>')
  else
    call CocAction('doHover')
  endif
endfunction

" Highlight symbol under cursor on CursorHold
autocmd CursorHold * silent call CocActionAsync('highlight')

" Remap for rename current word
" nmap <leader>rn <Plug>(coc-rename)

" Remap for format selected region
" xmap <leader>=  <Plug>(coc-format-selected)
" nmap <leader>=  <Plug>(coc-format-selected)

augroup mygroup
  autocmd!
  " Setup formatexpr specified filetype(s).
  autocmd FileType typescript,json setl formatexpr=CocAction('formatSelected')
  " Update signature help on jump placeholder
  autocmd User CocJumpPlaceholder call CocActionAsync('showSignatureHelp')
augroup end

" Remap for do codeAction of selected region, ex: `<leader>aap` for current paragraph
" xmap <leader>a  <Plug>(coc-codeaction-selected)
" nmap <leader>a  <Plug>(coc-codeaction-selected)

" Remap for do codeAction of current line
" nmap <leader>ac  <Plug>(coc-codeaction)
" Fix autofix problem of current line
" nmap <leader>qf  <Plug>(coc-fix-current)

" Create mappings for function text object, requires document symbols feature of languageserver.
" xmap if <Plug>(coc-funcobj-i)
" xmap af <Plug>(coc-funcobj-a)
" omap if <Plug>(coc-funcobj-i)
" omap af <Plug>(coc-funcobj-a)

" Use <C-d> for select selections ranges, needs server support, like: coc-tsserver, coc-python
" nmap <silent> <C-d> <Plug>(coc-range-select)
" xmap <silent> <C-d> <Plug>(coc-range-select)

" Use `:Format` to format current buffer
command! -nargs=0 Format :call CocAction('format')

" Use `:Fold` to fold current buffer
" command! -nargs=? Fold :call CocAction('fold', <f-args>)

" use `:OR` for organize import of current buffer
command! -nargs=0 OR :call CocAction('runCommand', 'editor.action.organizeImport')

" Add status line support, for integration with other plugin, checkout `:h coc-status`
set statusline^=%{coc#status()}%{get(b:,'coc_current_function','')}

" Using CocList
" Show all diagnostics
nnoremap <silent> <space>a  :<C-u>CocList diagnostics<cr>
" Manage extensions
" nnoremap <silent> <space>e  :<C-u>CocList extensions<cr>
" Show commands
" nnoremap <silent> <space>c  :<C-u>CocList commands<cr>
" Find symbol of current document
" nnoremap <silent> <space>o  :<C-u>CocList outline<cr>
" Search workspace symbols
" nnoremap <silent> <space>s  :<C-u>CocList -I symbols<cr>
" Do default action for next item.
" nnoremap <silent> <space>j  :<C-u>CocNext<CR>
" Do default action for previous item.
" nnoremap <silent> <space>k  :<C-u>CocPrev<CR>
" Resume latest coc list
" nnoremap <silent> <space>p  :<C-u>CocListResume<CR>







" " rename the current word in the cursor
" nmap <leader>vr  <Plug>(coc-rename)
" nmap <leader>vf  <Plug>(coc-format-selected)
" vmap <leader>vf  <Plug>(coc-format-selected)

" " run code actions
" vmap <leader>va  <Plug>(coc-codeaction-selected)
" nmap <leader>va  <Plug>(coc-codeaction-selected)

" " map <F2> :echo 'Current time is ' . strftime('%c')<CR>
" " map <F2> : <Plug>(coc-command-python.setInterpreter)
nnoremap <F2> :<C-u>CocCommand python.setInterpreter<CR>
nnoremap <F3> :<C-u>CocCommand python.setLinter<CR>

" set statusline+=%{coc#status()}
" set statusline^=%{coc#status()}

" hi! link CocErrorSign WarningMsg
" hi! link CocWarningSign Number
" hi! link CocInfoSign Type


" === fzf ===
let g:fzf_colors =
\ { 'fg':      ['fg', 'Normal'],
  \ 'bg':      ['bg', 'Normal'],
  \ 'hl':      ['fg', 'Comment'],
  \ 'fg+':     ['fg', 'CursorLine', 'CursorColumn', 'Normal'],
  \ 'bg+':     ['bg', 'CursorLine', 'CursorColumn'],
  \ 'hl+':     ['fg', 'Statement'],
  \ 'info':    ['fg', 'PreProc'],
  \ 'border':  ['fg', 'Ignore'],
  \ 'prompt':  ['fg', 'Conditional'],
  \ 'pointer': ['fg', 'Exception'],
  \ 'marker':  ['fg', 'Keyword'],
  \ 'spinner': ['fg', 'Label'],
  \ 'header':  ['fg', 'Comment'] }


" === fugitive ===
if exists("*fugitive#statusline")
  set statusline+=%{fugitive#statusline()}
endif


" === nerdtree ===
" autocmd vimenter * NERDTree
" Close vim if the only window left open is a NERDTree
autocmd BufEnter * if (winnr("$") == 1 && exists("b:NERDTree") && b:NERDTree.isTabTree()) | q | endif
" If more than one window and previous buffer was NERDTree, go back to it.
autocmd BufEnter * if bufname('#') =~# "^NERD_tree_" && winnr('$') > 1 | b# | endif


" === buffergator ===
let g:buffergator_suppress_keymaps = 1


" === bufexplorer ===
let g:bufExplorerDisableDefaultKeyMapping = 1
let g:bufExplorerShowRelativePath = 1
" let g:bufExplorerSortBy='extension'  " Sort by file extension.
let g:bufExplorerSortBy='fullpath'   " Sort by full file path name.
" let g:bufExplorerSortBy='mru'        " Sort by most recently used.
" let g:bufExplorerSortBy='name'       " Sort by the buffer's name.
" let g:bufExplorerSortBy='number'     " Sort by the buffer's number.
let g:bufExplorerSplitBelow=1        " Split new window above current.

" === vim-airline ===
let g:airline_theme = 'solarized'
let g:airline#extensions#syntastic#enabled = 1
let g:airline#extensions#branch#enabled = 1
let g:airline#extensions#tabline#enabled = 1
let g:airline#extensions#tagbar#enabled = 0
let g:airline_skip_empty_sections = 1

" coc
" let g:airline#extensions#disable_rtp_load = 1
" let g:airline_extensions = ['branch', 'hunks', 'coc']
" Configure error/warning section to use coc.nvim
let g:airline_section_error = '%{airline#util#wrap(airline#extensions#coc#get_error(),0)}'
let g:airline_section_warning = '%{airline#util#wrap(airline#extensions#coc#get_warning(),0)}'

" tmuxline
let g:tmuxline_powerline_separators = 0
" If you set airline theme manually, make sure the airline-tmuxline extension is disabled, so it doesn't overwrite the theme
let g:airline#extensions#tmuxline#enabled = 0

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


" === vista.vim ===
" How each level is indented and what to prepend.
" This could make the display more compact or more spacious.
" let g:vista_icon_indent = ["╰─▸ ", "├─▸ "]
let g:vista_icon_indent = ["╰─▸  ", "▸ "]

" Executive used when opening vista sidebar without specifying it.
" See all the avaliable executives via `:echo g:vista#executives`.
" :echo g:vista#executives
" ['ale', 'coc', 'ctags', 'lcn', 'vim_lsp']
let g:vista_default_executive = 'coc'

" Set the executive for some filetypes explicitly. Use the explicit executive
" instead of the default one for these filetypes when using `:Vista` without
" specifying the executive.
let g:vista_executive_for = {
  \ 'cpp': 'vim_lsp',
  \ 'php': 'vim_lsp',
  \ }

" Declare the command including the executable and options used to generate ctags output
" for some certain filetypes.The file path will be appened to your custom command.
" For example:
let g:vista_ctags_cmd = {
      \ 'haskell': 'hasktags -x -o - -c',
      \ }

" To enable fzf's preview window set g:vista_fzf_preview.
" The elements of g:vista_fzf_preview will be passed as arguments to fzf#vim#with_preview()
" For example:
let g:vista_fzf_preview = ['right:50%']


" Ensure you have installed some decent font to show these pretty symbols, then you can enable icon for the kind.
let g:vista#renderer#enable_icon = 1

" The default icons can't be suitable for all the filetypes, you can extend it as you wish.
let g:vista#renderer#icons = {
\   "function": "\uf794",
\   "variable": "\uf71b",
\  }

function! NearestMethodOrFunction() abort
  return get(b:, 'vista_nearest_method_or_function', '')
endfunction
set statusline+=%{NearestMethodOrFunction()}

" By default vista.vim never run if you don't call it explicitly.
" If you want to show the nearest function in your statusline automatically,
" you can add the following line to your vimrc
" autocmd VimEnter * call vista#RunForNearestMethodOrFunction()

" let g:vista_echo_cursor_strategy = 'both'
let g:vista_echo_cursor_strategy = 'scroll'
let g:vista_stay_on_open = 0
let g:vista_sidebar_width = 40

" autocmd! VimEnter * Vista coc<CR>
" nnoremap <F5> :Vista coc<CR>


" === tagbar ===
let g:tagbar_sort = 0
let g:tagbar_compact = 1
let g:tagbar_iconchars = ['▸', '▾']
let g:tagbar_width = 40
autocmd FileType c,cpp,py,sh nested :TagbarOpen
" autocmd FileType * nested :call tagbar#autoopen(0) 
" let g:tagbar_type_python  = {
" \ 'ctagstype' : 'python',
" \ 'kinds'     : [
" 	\ 'c:classes',
" 	\ 'f:functions',
" \ ]
" \ }
"	\ 'i:interfaces',
"	\ 'd:constant definitions',
" nnoremap <silent> <F5> :TagbarToggle<CR> 



" === vim-slime ===
let g:slime_target = "tmux"
let g:slime_python_ipython = 1
" let g:slime_default_config = {"socket_name": get(split($TMUX, ","), 0), "target_pane": ":.2"}
let g:slime_default_config = {"socket_name": "default", "target_pane": "{right-of}"}
" let g:slime_paste_file = "/tmp/slime/slime_paste"
let g:slime_paste_file = "~/slime_paste"


" === vim-multiple-cursors ===
let g:multi_cursor_use_default_mapping=0

" Default mapping
let g:multi_cursor_start_word_key      = '<C-n>'
let g:multi_cursor_select_all_word_key = '<A-n>'
let g:multi_cursor_start_key           = ''
let g:multi_cursor_select_all_key      = ''
let g:multi_cursor_next_key            = '<C-n>'
let g:multi_cursor_prev_key            = '<C-p>'
let g:multi_cursor_skip_key            = '<C-x>'
let g:multi_cursor_quit_key            = '<Esc>'


" }}}

" Commands {{{

" Jump to the last position when reopening a file
if has("autocmd")
  au BufReadPost * if line("'\"") > 1 && line("'\"") <= line("$") | exe "normal! g'\"" | endif
endif
autocmd BufRead * normal zz

" Reload icons after init source
if exists('g:loaded_webdevicons')
  call webdevicons#refresh()
endif

" Correct comment highlight in jsonc
autocmd FileType json syntax match Comment +\/\/.\+$+

" }}}

" Color Scheme {{{
" nnoremap <F12> :call CommentColorSchemeToggle()<CR>
let g:ccs_flag = 0
function! CommentColorSchemeToggle()
    if g:ccs_flag
		hi Comment guifg=#93a1a1
		hi Comment guibg=#fdf6e3
        let g:ccs_flag = 0
    else
		hi Comment guifg=#fdf6e3
		hi Comment guibg=#93a1a1
        let g:ccs_flag = 1
    endif
endfunction
silent! colorscheme solarized8
set background=light
set t_Co=256

highlight SignColumn guibg=#EEE8D5
highlight Pmenu ctermbg=None guibg=#E8E2D1
highlight CocFloating ctermfg=None  guibg=#E8E2D1

" split bar styling
set fillchars+=vert:\ 
hi VertSplit ctermbg=NONE guibg=NONE
hi VertSplit ctermfg=NONE guifg=NONE

" }}}

function! StatusDiagnostic() abort
	let info = get(b:, 'coc_diagnostic_info', {})
	if empty(info) | return '' | endif
	let msgs = []
	if get(info, 'error', 0)
		call add(msgs, 'E' . info['error'])
	endif
	if get(info, 'warning', 0)
		call add(msgs, 'W' . info['warning'])
	endif
	return join(msgs, ' ') . ' ' . get(g:, 'coc_status', '')
endfunction
set statusline+=%{StatusDiagnostic()}

" Folding {{{
augroup filetype_vim
	autocmd!
    autocmd FileType vim setlocal foldmethod=marker
augroup END

set modelines=1

" :CocCommand python.workspaceSymbols.rebuildOnFileSave = false
" :checkhealth
" :CocConfig
" :CocInstall coc-snippets
" :CocCommand snippets.editSnippets
" :CocInstall coc-python
" :CocInfo
" :Vista finder coc
" F2, F3, F4, F5, F12
" so $VIMRUNTIME/syntax/hitest.vim
" CocInstall coc-python
" CocInstall coc-git
" https://github.com/ryanoasis/nerd-fonts/blob/master/patched-fonts/UbuntuMono/Regular/complete/Ubuntu%20Mono%20Nerd%20Font%20Complete.ttf
" space t
" space f
" space F
" CocCommand explorer
"
"
" :CocList extensions
" :CocList commands
" https://github.com/neoclide/coc.nvim/wiki/Using-coc-extensions
" An example config to use the custom command Tsc for tsserver.watchBuild:
" command! -nargs=0 Tsc    :CocCommand tsserver.watchBuild
"
" :CocInstall coc-json coc-css
"
" After adding this to your vimrc run PlugInstall. This has the limitation that you can't uninstall the extension by using :CocUninstall and that automatic update support is not available.
" :CocUpdate
" :CocUninstall coc-css
"
"conda install ptpython
"pip install isort
"pip install ptpython --upgrade
"pip install ptipython

" :Isort
" monkeytype
" mypy
" :gd go to definition
" ctags -R --language=python --exclude=tmp --exclude=output
" ]m  goto end of method
" ctrl+w s  split
" ctrl+w o  only window
" 
"

"python.pythonPath":"/home/jason/python", in coc-settings.json
""coc.preferences.formatOnSaveFiletypes": ["python"],
" pip install 'python-language-server[all]'


" }}}

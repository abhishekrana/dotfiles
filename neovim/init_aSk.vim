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
Plug 'neoclide/coc.nvim', {'tag': '*', 'do': './install.sh'}
Plug 'neoclide/coc-json', {'do': 'yarn install --frozen-lockfile'}
Plug 'neoclide/coc-python', {'do': 'yarn install --frozen-lockfile'}
Plug 'neoclide/coc-emmet', {'do': 'yarn install --frozen-lockfile'}
Plug 'neoclide/coc-pairs', {'do': 'yarn install --frozen-lockfile'}
Plug 'neoclide/coc-lists', {'do': 'yarn install --frozen-lockfile'}
Plug 'neoclide/coc-snippets', {'do': 'yarn install --frozen-lockfile'}

" Fuzzy file finding, project searching, file browsing
Plug 'junegunn/fzf', { 'dir': '~/.fzf', 'do': './install --all' }
Plug 'junegunn/fzf.vim'
Plug 'scrooloose/nerdtree'

" Git
Plug 'tpope/vim-fugitive'
Plug 'airblade/vim-gitgutter'
Plug 'tpope/vim-rhubarb'

" UI
Plug 'vim-airline/vim-airline'
Plug 'vim-airline/vim-airline-themes'
Plug 'edkolev/tmuxline.vim'
Plug 'lifepillar/vim-solarized8'
Plug 'altercation/vim-colors-solarized'
Plug 'ryanoasis/vim-devicons'

" Vim-Session
Plug 'xolox/vim-misc'
Plug 'xolox/vim-session'

" Misc
Plug 'machakann/vim-highlightedyank'
Plug 'tpope/vim-commentary'
Plug 'liuchengxu/vista.vim'

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
noremap <leader>f :Files<CR>
noremap <leader>F :GFiles<CR>
nmap <leader>t :BTags<CR>
nmap <leader>T :Tags<CR>
nnoremap <silent><leader>s :Rg <C-R>=expand("<cword>")<CR><CR>
nnoremap <silent><leader>S :Ag <C-R>=expand("<cword>")<CR><CR>

" Custom Mappings
noremap ff <Esc>:w<CR>
inoremap ff <Esc>:w<CR>
noremap <S-l> <ESC>$
noremap <S-h> <ESC>^
noremap <S-j> <ESC>:call cursor(0, len(getline('.'))/2)<cr> 
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
noremap <leader>w :windo bd<CR>
noremap <leader>q :SaveSession! .session<CR>:qa<CR>
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
nnoremap <leader>z :OpenSession! .session<cr>
" autocmd! VimEnter * OpenSession! .session<cr>

" Edit Vimrc
nnoremap <leader>ev :vsplit $MYVIMRC<cr>

"nnoremap <leader>F :call QuickfixToggle()<cr>
"nnoremap <leader>i :call PasteToggle()<cr>
"nnoremap <leader>= :call BackgroundToggle()<cr>
"noremap <leader>| :!ctags -R --c++-kinds=+p --fields=+iaSl --extra=+q; find . -name "*.c" -o -name "*.cpp" -o -name "*.h" -o -name "*.hpp" > cscope.files; cscope -q -R -b -i cscope.files <CR>
"noremap <leader>\ :!ctags -R --fields=+iaSl --extra=+q<CR>

" }}}

" Plugins Config {{{

" === vim-session ===
let g:session_autosave = 'no'
let g:session_directory='.'

" === coc ===

" If hidden is not set, TextEdit might fail. (Hides buffers instead of closing them)
set hidden

" Some servers have issues with backup files, see #649
set nobackup
set nowritebackup

" Better display for messages
set cmdheight=1

" Smaller updatetime for CursorHold & CursorHoldI
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

" " Use <cr> to confirm completion, `<C-g>u` means break undo chain at current position.
" " Coc only does snippet and additional edit on confirm.
" inoremap <expr> <cr> pumvisible() ? "\<C-y>" : "\<C-g>u\<CR>"

" " Use `[c` and `]c` to navigate diagnostics
" nmap <silent> [c <Plug>(coc-diagnostic-prev)
" nmap <silent> ]c <Plug>(coc-diagnostic-next)

" " Remap keys for gotos
" nmap <silent> gd <Plug>(coc-definition)
" nmap <silent> gy <Plug>(coc-type-definition)
" nmap <silent> gi <Plug>(coc-implementation)
" nmap <silent> gr <Plug>(coc-references)
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

" " Remap for rename current word
" nmap <leader>rn <Plug>(coc-rename)

" " Remap for format selected region
" xmap <leader>f  <Plug>(coc-format-selected)
" nmap <leader>f  <Plug>(coc-format-selected)

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
" Fix autofix problem of current line; 
" nmap <leader>qf  <Plug>(coc-fix-current) " This mappling slows file closing

" " Use `:Format` to format current buffer
" command! -nargs=0 Format :call CocAction('format')

" " Use `:Fold` to fold current buffer
" command! -nargs=? Fold :call     CocAction('fold', <f-args>)

" " Using CocList
" " Show all diagnostics
nnoremap <silent> <space>a  :<C-u>CocList diagnostics<cr>
" " Manage extensions
" nnoremap <silent> <space>e  :<C-u>CocList extensions<cr>
" " Show commands
" nnoremap <silent> <space>c  :<C-u>CocList commands<cr>
" " Find symbol of current document
" nnoremap <silent> <space>o  :<C-u>CocList outline<cr>
" Search workspace symbols
" nnoremap <silent> <space>s  :<C-u>CocList -I symbols<cr>
" " Do default action for next item.
" nnoremap <silent> <space>j  :<C-u>CocNext<CR>
" " Do default action for previous item.
" nnoremap <silent> <space>k  :<C-u>CocPrev<CR>
" " Resume latest coc list
" nnoremap <silent> <space>p  :<C-u>CocListResume<CR>
"

" set statusline+=%{coc#status()}
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
" Close vim if the only window left open is a NERDTree
autocmd BufEnter * if (winnr("$") == 1 && exists("b:NERDTree") && b:NERDTree.isTabTree()) | q | endif


" === vim-airline ===
let g:airline_theme = 'solarized'
let g:airline#extensions#syntastic#enabled = 1
let g:airline#extensions#branch#enabled = 1
let g:airline#extensions#tabline#enabled = 1
let g:airline#extensions#tagbar#enabled = 1
let g:airline_skip_empty_sections = 1

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

" }}}

" Folding {{{
augroup filetype_vim
	autocmd!
    autocmd FileType vim setlocal foldmethod=marker
augroup END

set modelines=1

" }}}

" === colorscheme ===
" colorscheme solarized8
set background=light

" checkhealth


package commands

import (
	"fmt"
)

const bashCompletion = `# bash completion for gtd
_gtd() {
  local cur prev words cword
  _init_completion || return
  local cmds="add list view update done delete inbox today review search sync config subtask depends serve completion remarkable help"
  if [[ ${cword} -eq 1 ]]; then
    COMPREPLY=( $(compgen -W "${cmds}" -- "${cur}") )
    return
  fi
}
complete -F _gtd gtd
`

const zshCompletion = `#compdef gtd
_gtd() {
  local -a commands
  commands=(
    'add:Add a task'
    'list:List tasks'
    'view:View task details'
    'update:Update a task'
    'done:Mark task complete'
    'delete:Delete a task'
    'inbox:Show inbox'
    'today:Show today tasks'
    'review:Weekly review'
    'search:Search tasks'
    'sync:Sync repository'
    'config:Update config'
    'subtask:Manage subtasks'
    'depends:Manage dependencies'
    'serve:Start HTTP API'
    'completion:Print shell completion'
    'remarkable:ReMarkable sync utilities'
    'help:Show help'
  )
  _describe 'command' commands
}
compdef _gtd gtd
`

// Completion prints shell completion scripts.
func Completion(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("shell required: bash|zsh")
	}
	switch args[0] {
	case "bash":
		fmt.Print(bashCompletion)
	case "zsh":
		fmt.Print(zshCompletion)
	default:
		return fmt.Errorf("unknown shell: %s", args[0])
	}
	return nil
}

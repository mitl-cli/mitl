package commands

import (
	"fmt"
	"os"
	"strings"
)

// Completion provides shell completion scripts for bash and zsh.
// Usage:
//
//	mitl completion           # prints completions for all supported shells
//	mitl completion bash      # prints bash completion
//	mitl completion zsh       # prints zsh completion
func Completion(args []string) error {
	shell := ""
	if len(args) > 0 {
		shell = strings.ToLower(args[0])
	}

	switch shell {
	case "bash":
		printBashCompletion()
		return nil
	case "zsh":
		printZshCompletion()
		return nil
	case "", "all":
		// Print both so Homebrew's generator can detect them
		printBashCompletion()
		fmt.Println()
		printZshCompletion()
		return nil
	default:
		fmt.Fprintf(os.Stderr, "unknown shell: %s (supported: bash, zsh)\n", shell)
		return fmt.Errorf("unsupported shell: %s", shell)
	}
}

func printBashCompletion() {
	// Simple bash completion that suggests top-level commands and flags
	fmt.Println(`# bash completion for mitl
_mitl_completions()
{
    local cur prev words cword
    _init_completion || return

    local -a commands
    commands=(
        analyze digest hydrate run shell inspect setup runtime doctor cache volumes bench completion help version
    )

    case ${COMP_CWORD} in
        1)
            COMPREPLY=( $(compgen -W "${commands[*]}" -- "$cur") )
            return ;;
        *)
            case ${COMP_WORDS[1]} in
                run)
                    COMPREPLY=( $(compgen -W "--verbose --debug" -- "$cur") ) ;;
                bench)
                    COMPREPLY=( $(compgen -W "run compare list export --iterations --category --compare --output --format --parallel --verbose" -- "$cur") ) ;;
                completion)
                    COMPREPLY=( $(compgen -W "bash zsh" -- "$cur") ) ;;
                *)
                    COMPREPLY=( $(compgen -W "--verbose --debug" -- "$cur") ) ;;
            esac
            return ;;
    esac
}
complete -F _mitl_completions mitl`)
}

func printZshCompletion() {
	fmt.Println(`#compdef mitl
_mitl() {
  local -a commands
  commands=(
    'analyze:Analyze host toolchains'
    'digest:Calculate and inspect project digests'
    'hydrate:Build project capsule'
    'run:Run command in capsule'
    'shell:Open shell in capsule'
    'inspect:Analyze project and show Dockerfile'
    'setup:Setup default runtime'
    'runtime:Runtime info/benchmark/recommend'
    'doctor:System health check'
    'cache:Cache management'
    'volumes:Volume management'
    'bench:Run benchmarks and performance comparisons'
    'completion:Generate shell completion scripts'
    'version:Show version'
    'help:Show help'
  )

  _arguments \
    '1: :->cmds' \
    '*:: :->args'

  case $state in
    cmds)
      _describe 'command' commands
      ;;
    args)
      case $words[1] in
        completion)
          _values 'shell' bash zsh
          ;;
        bench)
          _values 'options' run compare list export --iterations --category --compare --output --format --parallel --verbose
          ;;
        *)
          _message 'arguments'
          ;;
      esac
      ;;
  esac
}
_mitl "$@"`)
}

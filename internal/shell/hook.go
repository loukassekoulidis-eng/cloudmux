package shell

import (
	"bytes"
	"fmt"
	"text/template"
)

var bashHook = `# cloudmux shell hook (bash)
cloudmux() {
    local cmd="${1:-}"
    case "$cmd" in
        use|logout)
            local output
            output="$(command {{.Binary}} "$@")"
            local rc=$?
            if [ $rc -eq 0 ]; then
                eval "$output"
            else
                echo "$output" >&2
                return $rc
            fi
            ;;
        *)
            command {{.Binary}} "$@"
            ;;
    esac
}

# Prompt integration (skipped if Starship is active — use a custom Starship module instead)
if [ -z "${STARSHIP_SHELL:-}" ]; then
    _cloudmux_prompt() {
        if [ -n "${CLOUDMUX_ACTIVE_PROFILE:-}" ]; then
            echo "[cloudmux: ${CLOUDMUX_ACTIVE_PROFILE}] "
        fi
    }
    if [[ ! "$PS1" == *'_cloudmux_prompt'* ]]; then
        PS1='$(_cloudmux_prompt)'"$PS1"
    fi
fi
`

var zshHook = `# cloudmux shell hook (zsh)
cloudmux() {
    local cmd="${1:-}"
    case "$cmd" in
        use|logout)
            local output
            output="$(command {{.Binary}} "$@")"
            local rc=$?
            if [ $rc -eq 0 ]; then
                eval "$output"
            else
                echo "$output" >&2
                return $rc
            fi
            ;;
        *)
            command {{.Binary}} "$@"
            ;;
    esac
}

# Prompt integration (skipped if Starship is active — use a custom Starship module instead)
if [[ -z "${STARSHIP_SHELL:-}" ]]; then
    _cloudmux_prompt() {
        if [[ -n "${CLOUDMUX_ACTIVE_PROFILE:-}" ]]; then
            echo "[cloudmux: ${CLOUDMUX_ACTIVE_PROFILE}] "
        fi
    }
    if [[ ! "$PROMPT" == *'_cloudmux_prompt'* ]]; then
        PROMPT='$(_cloudmux_prompt)'"$PROMPT"
    fi
fi
`

var fishHook = `# cloudmux shell hook (fish)
function cloudmux
    set -l cmd $argv[1]
    switch "$cmd"
        case use logout
            set -l output (command {{.Binary}} $argv 2>/dev/null)
            set -l rc $status
            if test $rc -eq 0
                for line in $output
                    # Convert POSIX export/unset to fish syntax
                    if string match -qr '^export ([^=]+)='\''(.*)'\''$' -- $line
                        set -l matches (string match -r '^export ([^=]+)='\''(.*)'\''$' -- $line)
                        set -gx $matches[2] $matches[3]
                    else if string match -qr '^unset (.+)$' -- $line
                        set -l matches (string match -r '^unset (.+)$' -- $line)
                        set -e $matches[2]
                    end
                end
            else
                command {{.Binary}} $argv
                return $rc
            end
        case '*'
            command {{.Binary}} $argv
    end
end

# Prompt integration (skipped if Starship is active)
if not set -q STARSHIP_SHELL
    function _cloudmux_prompt --on-event fish_prompt
        if set -q CLOUDMUX_ACTIVE_PROFILE
            echo -n "[cloudmux: $CLOUDMUX_ACTIVE_PROFILE] "
        end
    end
end
`

var hooks = map[string]string{
	"bash": bashHook,
	"zsh":  zshHook,
	"fish": fishHook,
}

type hookData struct {
	Binary string
}

func GenerateHook(shellType string, binary string) (string, error) {
	tmplStr, ok := hooks[shellType]
	if !ok {
		return "", fmt.Errorf("unsupported shell %q (supported: bash, zsh, fish)", shellType)
	}

	tmpl, err := template.New(shellType).Parse(tmplStr)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, hookData{Binary: binary}); err != nil {
		return "", err
	}

	return buf.String(), nil
}

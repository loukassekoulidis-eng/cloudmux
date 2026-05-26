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

# Prompt integration
_cloudmux_prompt() {
    if [ -n "${CLOUDMUX_ACTIVE_PROFILE:-}" ]; then
        echo "[cloudmux: ${CLOUDMUX_ACTIVE_PROFILE}] "
    fi
}

if [[ ! "$PS1" == *'_cloudmux_prompt'* ]]; then
    PS1='$(_cloudmux_prompt)'"$PS1"
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

# Prompt integration
_cloudmux_prompt() {
    if [[ -n "${CLOUDMUX_ACTIVE_PROFILE:-}" ]]; then
        echo "[cloudmux: ${CLOUDMUX_ACTIVE_PROFILE}] "
    fi
}

if [[ ! "$PROMPT" == *'_cloudmux_prompt'* ]]; then
    PROMPT='$(_cloudmux_prompt)'"$PROMPT"
fi
`

var hooks = map[string]string{
	"bash": bashHook,
	"zsh":  zshHook,
}

type hookData struct {
	Binary string
}

func GenerateHook(shellType string, binary string) (string, error) {
	tmplStr, ok := hooks[shellType]
	if !ok {
		return "", fmt.Errorf("unsupported shell %q (supported: bash, zsh)", shellType)
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

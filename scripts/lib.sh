# shellcheck shell=bash
# Sourced by the scripts beside it. Leaves the shell at the repository root.

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$root" || exit 1

# bun installs itself in ~/.bun/bin, which non-interactive shells often miss.
[[ -d "$HOME/.bun/bin" ]] && PATH="$HOME/.bun/bin:$PATH"

die() { echo "error: $*" >&2; exit 1; }

# need <command> <hint>
need() { command -v "$1" >/dev/null || die "$1 is missing — $2"; }

# load_env exports .env the way the server reads it (viper's dotenv parser),
# so a script and the server never see two different values: variables
# already in the environment (and not empty) win, and otherwise the last line
# for a name does.
#
# The line is cut at its first "=" by hand. `IFS='=' read` would drop a lone
# trailing "=", which every base64 VARD_SECRET_KEY ends with: the scripts
# then ran the server with a different key from the one `make run` reads.
# core/config's tests hold this function to the server's reading.
load_env() {
	[[ -f .env ]] || return 0
	local line key value inherited=" "
	local pattern='^[[:space:]]*(export[[:space:]]+)?([A-Za-z_][A-Za-z0-9_]*)[[:space:]]*=[[:space:]]*(.*)$'
	local double='^"(.*)"[[:space:]]*(#.*)?$' single="^'(.*)'[[:space:]]*(#.*)?$"

	# What the environment held before .env: that is what wins.
	while IFS= read -r line || [[ -n $line ]]; do
		[[ $line =~ $pattern ]] || continue
		key="${BASH_REMATCH[2]}"
		[[ -n ${!key:-} ]] && inherited+="$key "
	done <.env

	while IFS= read -r line || [[ -n $line ]]; do
		# A .env saved with Windows line endings ends every line with a
		# carriage return; a DSN carrying it fails to parse with a message
		# about sslmode, which sends you looking in the wrong file.
		line="${line%$'\r'}"
		[[ $line =~ $pattern ]] || continue
		key="${BASH_REMATCH[2]}" value="${BASH_REMATCH[3]}"
		[[ $inherited == *" $key "* ]] && continue

		if [[ $value =~ $double || $value =~ $single ]]; then
			value="${BASH_REMATCH[1]}"
		else
			# Unquoted: a comment starts at " #"; trailing spaces go.
			value="${value%%[[:space:]]#*}"
			value="${value%"${value##*[![:space:]]}"}"
		fi

		export "$key=$value"
	done <.env
}

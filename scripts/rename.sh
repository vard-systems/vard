#!/usr/bin/env bash
#
# Rename the project: scripts/rename.sh "Acme ID" acme (make rename).
#
# Each application keeps its own brand: internal/brand for the server, and
# src/lib/brand.ts in each app. This rewrites all three, and what cannot read
# them and spells the name itself: the Go module path in go.mod and every
# import, cmd/<slug>/, the VARD_-style environment prefix, the scripts,
# deploy/, CI and the docs.
#
# It edits tracked files only, and only in a clean tree, so `git diff` is the
# whole of what it did and `git checkout .` undoes it. It never touches .env:
# rename the variables in it yourself.

source "$(dirname "$0")/lib.sh"

[[ $# -eq 2 ]] || die 'usage: scripts/rename.sh "New Name" newslug'
new_name="$1" new_slug="$2"

# The slug becomes cookie names, a Redis prefix, URNs, Postgres names and the
# environment's prefix; internal/brand's tests refuse anything else.
[[ "$new_slug" =~ ^[a-z][a-z0-9]*$ ]] || die "the slug is lower-case letters and digits, starting with a letter"
# The name is written into a Go string and a TypeScript one.
[[ -n "${new_name// /}" && "$new_name" != *[\"\'\\]* ]] || die "the name is not empty and has no quote or backslash"

[[ -z "$(git status --porcelain)" ]] || die "commit or stash your changes first, so the rename is a diff of its own"

field() { sed -n "s/^[[:space:]]*$1 = \"\(.*\)\"$/\1/p" internal/brand/brand.go; }
old_name="$(field Name)" old_slug="$(field Slug)"
[[ -n "$old_name" && -n "$old_slug" ]] || die "internal/brand/brand.go has no Name or Slug"
[[ "$old_slug" != "$new_slug" || "$old_name" != "$new_name" ]] || die "it is already called $new_name ($new_slug)"

# internal/brand's tests fail on the name written outside the brands, so a
# name the code already holds for something else (a fixture's
# acme.example.com) would fail them after the rename. These are the files they
# read.
code=('*.go' '*.ts' '*.svelte' '*.js' '*.json' '*.html' '*.css' '*.svg' ':!internal/brand' ':!web/*/src/lib/brand.ts')
for word in "$new_slug" "$new_name"; do
	if [[ "$word" != "$old_slug" && "$word" != "$old_name" ]] &&
		git grep -q -F -e "$word" -- "${code[@]}"; then
		die "\"$word\" is already in the code: $(git grep -l -F -e "$word" -- "${code[@]}" | head -5 | tr '\n' ' ')"
	fi
done

old_upper="$(tr '[:lower:]' '[:upper:]' <<<"$old_slug")"
new_upper="$(tr '[:lower:]' '[:upper:]' <<<"$new_slug")"

echo "==> $old_name ($old_slug) → $new_name ($new_slug)"

# One pass over each line for all three spellings (the environment prefix,
# the name, the slug), so nothing a replacement wrote is replaced again: a new
# name holding the old slug stays as it was given. The values reach perl
# through the environment, so nothing in them is read as a pattern.
git grep -lzI -i -e "$old_slug" -- . | OLD_UPPER="$old_upper" NEW_UPPER="$new_upper" \
	OLD_NAME="$old_name" NEW_NAME="$new_name" OLD_SLUG="$old_slug" NEW_SLUG="$new_slug" \
	xargs -0 perl -pi -e '
		BEGIN { %to = ($ENV{OLD_UPPER} => $ENV{NEW_UPPER}, $ENV{OLD_NAME} => $ENV{NEW_NAME}, $ENV{OLD_SLUG} => $ENV{NEW_SLUG}) }
		s/(\Q$ENV{OLD_UPPER}\E|\Q$ENV{OLD_NAME}\E|\Q$ENV{OLD_SLUG}\E)/$to{$1}/g;
	'

if [[ "$old_slug" != "$new_slug" && -d "cmd/$old_slug" ]]; then
	git mv "cmd/$old_slug" "cmd/$new_slug"
fi

# gofmt sorts imports by path, which the module's new name may reorder.
git ls-files -z '*.go' | xargs -0 gofmt -w

# A new name may hold the old slug on purpose; those lines are not leftovers.
left="$(git grep -n -i -e "$old_slug" -- . | grep -v -F -e "$new_name" || true)"
if [[ -n "$left" ]]; then
	echo "==> still spelled the old way, in a case this does not rewrite:"
	echo "$left"
fi

cat <<EOF
==> done: review it with git diff, then run make check.
    Still yours to do:
    - the console's links (web/console/src/lib/brand.ts), if the repository moved
    - a new mark: each favicon (internal/brand/favicon.svg,
      web/*/src/lib/assets/favicon.svg) and icon (web/*/src/lib/brand.ts)
    - .env: rename every ${old_upper}_ variable to ${new_upper}_ (this never reads .env)
EOF
if [[ "$old_slug" != "$new_slug" ]]; then
	cat <<EOF
    - an installation already running: the slug names its cookies (everybody is
      signed out), its Redis keys, its system APIs' identifiers and the exported
      login flows' \$schema, so a new slug is for before the first deployment
EOF
fi

#!/bin/zsh
# set -euo pipefail
export GO111MODULE=on
export PROJECT_DIR="$(pwd)"

alias jq="jq --unbuffered"

AUTHORS=(filecoin-project)

echo "project dir: ${PROJECT_DIR}"
echo "arthurs: ${AUTHORS}"

[[ -n "${REPO_FILTER+x}" ]] || REPO_FILTER="github.com/(${$(printf "|%s" "${AUTHORS[@]}"):1})"

[[ -n "${IGNORED_FILES+x}" ]] || IGNORED_FILES='^\(\.gx\|package\.json\|\.travis\.yml\|go.mod\|go\.sum|\.github|\.circleci\)$'

NL=$'\n'

ROOT_DIR="$(git rev-parse --show-toplevel)"

msg() {
    echo "$*" >&2
}

statlog() {
    local module="$1"
    local rpath="$PROJECT_DIR/$(strip_version "$module")"
    local start="${2:-}"
    local end="${3:-HEAD}"
    local mailmap_file="$rpath/.mailmap"
    if ! [[ -e "$mailmap_file" ]]; then
        mailmap_file="$ROOT_DIR/.mailmap"
    fi

    local stack=()
    git -C "$rpath" -c mailmap.file="$mailmap_file" log --use-mailmap --shortstat --no-merges --pretty="tformat:%H%x09%aN%x09%aE" "$start..$end" | while read -r line; do
        if [[ -n "$line" ]]; then
            stack+=("$line")
            continue
        fi

        read -r changes

        changed=0
        insertions=0
        deletions=0
        while read count event; do
            if [[ "$event" =~ ^file ]]; then
                changed=$count
            elif [[ "$event" =~ ^insertion ]]; then
                insertions=$count
            elif [[ "$event" =~ ^deletion ]]; then
                deletions=$count
            else
                echo "unknown event $event" >&2
                exit 1
            fi
        done<<<"${changes//,/$NL}"

        for author in "${stack[@]}"; do
            IFS=$'\t' read -r hash name email <<<"$author"
            jq -n \
               --arg "hash" "$hash" \
               --arg "name" "$name" \
               --arg "email" "$email" \
               --argjson "changed" "$changed" \
               --argjson "insertions" "$insertions" \
               --argjson "deletions" "$deletions" \
               '{Commit: $hash, Author: $name, Email: $email, Files: $changed, Insertions: $insertions, Deletions: $deletions}'
        done
        stack=()
    done
}

# Returns a stream of deps changed between $1 and $2.
dep_changes() {
    {
        <"$1"
        <"$2"
    } | jq -s 'JOIN(INDEX(.[0][]; .Path); .[1][]; .Path; {Path: .[0].Path, Old: (.[1] | del(.Path)), New: (.[0] | del(.Path))}) | select(.New.Version != .Old.Version)'
}

# resolve_commits resolves a git ref for each version.
resolve_commits() {
    jq '. + {Ref: (.Version|capture("^((?<ref1>.*)\\+incompatible|v.*-(0\\.)?[0-9]{14}-(?<ref2>[a-f0-9]{12})|(?<ref3>v.*))$") | .ref1 // .ref2 // .ref3)}'
}

pr_link() {
    local repo="$1"
    local prnum="$2"
    local ghname="${repo##github.com/}"
    printf -- "[%s#%s](https://%s/pull/%s)" "$ghname" "$prnum" "$repo" "$prnum"
}

# Generate a release log for a range of commits in a single repo.
release_log() {
    setopt local_options BASH_REMATCH

    local module="$1"
    local start="$2"
    local end="${3:-HEAD}"
    local repo="$(strip_version "$1")"
    local dir="$PROJECT_DIR/"

    local commit pr
    git -C "$dir" log \
        --format='tformat:%H %s' \
        --first-parent \
        "$start..$end" |
        while read commit subject; do
            # Skip gx-only PRs.
            git -C "$dir" diff-tree --no-commit-id --name-only "$commit^" "$commit" |
                grep -v "${IGNORED_FILES}" >/dev/null || continue

            if [[ "$subject" =~ '^Merge pull request #([0-9]+) from' ]]; then
                local prnum="${BASH_REMATCH[2]}"
                local desc="$(git -C "$dir" show --summary --format='tformat:%b' "$commit" | head -1)"
                printf -- "- %s (%s)\n" "$desc" "$(pr_link "$repo" "$prnum")"
            elif [[ "$subject" =~ '\(#([0-9]+)\)$' ]]; then
                local prnum="${BASH_REMATCH[2]}"
                printf -- "- %s (%s)\n" "$subject" "$(pr_link "$repo" "$prnum")"
            else
                printf -- "- %s\n" "$subject"
            fi
        done
}

indent() {
    sed -e 's/^/  /'
}

mod_deps() {
    go list -mod=mod -json -m all | jq 'select(.Version != null)'
}

ensure() {
    local repo="$(strip_version "$1")"
    local commit="$2"
    local rpath="$PROJECT_DIR/$repo"
    if [[ ! -d "$rpath" ]]; then
        msg "Cloning $repo..."
        git clone "http://$repo" "$rpath" >&2
    fi

    if ! git -C "$rpath" rev-parse --verify "$commit" >/dev/null; then
        msg "Fetching $repo..."
        git -C "$rpath" fetch --all >&2
    fi

    git -C "$rpath" rev-parse --verify "$commit" >/dev/null || return 1
}

statsummary() {
    jq -s 'group_by(.Author)[] | {Author: .[0].Author, Commits: (. | length), Insertions: (map(.Insertions) | add), Deletions: (map(.Deletions) | add), Files: (map(.Files) | add)}' |
        jq '. + {Lines: (.Deletions + .Insertions)}'
}

strip_version() {
    local repo="$1"
    if [[ "$repo" =~ '.*/v[0-9]+$' ]]; then
        repo="$(dirname "$repo")"
    fi
    echo "$repo"
}

recursive_release_log() {
    local start="${1:-$(git tag -l | sort -V | grep -v -- '-rc' | grep 'v'| tail -n1)}"
    local end="${2:-$(git rev-parse HEAD)}"
    local repo_root="$(git rev-parse --show-toplevel)"
    local module="$(go list -m)"
    local dir="$(go list -m -f '{{.Dir}}')"

    echo "+++ start: ${start} +++"
    echo "+++ end: ${end} +++"
    echo "+++ repo_root: ${repo_root} +++"

    # if [[ "${GOPATH}/${module}" -ef "${dir}" ]]; then
    #    echo "This script requires the target module and all dependencies to live in a PROJECT_DIR."
    #    return 1
    # fi

    (
        local result=0
        local workspace="$(mktemp -d)"
        trap "$(printf 'rm -rf "%q"' "$workspace")" INT TERM EXIT
        cd "$workspace"

        mkdir extern
        ln -s "$repo_root"/extern/filecoin-ffi extern/filecoin-ffi
        ln -s "$repo_root"/extern/test-vectors extern/test-vectors

        # echo "Computing old deps..." >&2
        # git -C "$repo_root" show "$start:go.mod" >go.mod
        # mod_deps | resolve_commits | jq -s > old_deps.json

        # echo "Computing new deps..." >&2
        # git -C "$repo_root" show "$end:go.mod" >go.mod
        # mod_deps | resolve_commits | jq -s > new_deps.json

        rm -f go.mod go.sum

        printf -- "Generating Changelog for %s %s..%s\n" "$module" "$start" "$end" >&2

        printf -- "- %s:\n" "$module"
        release_log "$module" "$start" "$end" | indent


        statlog "$module" "$start" "$end" > statlog.json

        # dep_changes old_deps.json new_deps.json |
        #     jq --arg filter "$REPO_FILTER" 'select(.Path | match($filter))' |
        #     # Compute changelogs
        #     jq -r '"\(.Path) \(.New.Version) \(.New.Ref) \(.Old.Version) \(.Old.Ref // "")"' |
        #     while read module new new_ref old old_ref; do
        #         if ! ensure "$module" "$new_ref"; then
        #             result=1
        #             local changelog="failed to fetch repo"
        #         else
        #             statlog "$module" "$old_ref" "$new_ref" >> statlog.json
        #             local changelog="$(release_log "$module" "$old_ref" "$new_ref")"
        #         fi
        #         if [[ -n "$changelog" ]]; then
        #             printf -- "- %s (%s -> %s):\n" "$module" "$old" "$new"
        #             echo "$changelog" | indent
        #         fi
        #     done

        echo
        echo "Contributors"
        echo

        echo "| Contributor | Commits | Lines ± | Files Changed |"
        echo "|-------------|---------|---------|---------------|"
        statsummary <statlog.json |
            jq -s 'sort_by(.Lines) | reverse | .[]' |
            jq -r '"| \(.Author) | \(.Commits) | +\(.Insertions)/-\(.Deletions) | \(.Files) |"'
        return "$status"
    )
}

recursive_release_log "$@"


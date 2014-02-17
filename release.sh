#!/bin/sh
set -e 

TAG=$1
REPO=$2

if [ -z "$REPO" ]
then
	echo "$0 tag repo files..."
	exit 1
fi
shift 2
FILES=$@

if [ ! -e "$HOME/.github-autorelease" ]
then
  cat <<EOF
Please create a 'Personal Access Token' here: \
https://github.com/settings/applications
and write it to to ~/.github-autorelease"
EOF
  exit 1
fi
TOKEN=`cat $HOME/.github-autorelease`

RELEASE_JS='{
  "tag_name":"'$TAG'",
  "name":"'$TAG'",
  "body":"Release '$TAG'",
  "prerelease":true
}'

github() {
  curl -s -u "$TOKEN:x-oauth-basic" "$@"
}

echo "Creating release"
RESP=`github -X POST \
             https://api.github.com/repos/$REPO/releases \
             --data-binary "$RELEASE_JS"`

UPLOAD_URL=`echo "$RESP" | jq -r .upload_url | sed 's/{?name}//'`

if [ "$UPLOAD_URL" = "null" ]
then
  echo "ERROR: Couldn't create release:"
  echo "$RESP" | jq .
  exit 1
fi

echo "Uploading files:"
for file in $FILES
do
  echo "- $file"
  TMP=`mktemp`
  gzip -c $file > "$TMP"
  github -H 'Content-type: application/gzip' "$UPLOAD_URL?name=$(basename $file).gz" --data-binary @$TMP | jq -r .state
  rm $TMP
done


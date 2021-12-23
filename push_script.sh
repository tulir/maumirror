#!/bin/bash
if [[ ! -d $MM_REPOSITORY_OWNER ]]; then
	echo "Creating $(pwd)/$MM_REPOSITORY_OWNER"
	mkdir $MM_REPOSITORY_OWNER
fi
cd $MM_REPOSITORY_OWNER
if [[ ! -z "$MM_SOURCE_KEY_PATH" ]]; then
	export GIT_SSH_COMMAND="ssh -o StrictHostKeyChecking=no -i $MM_SOURCE_KEY_PATH"
	SOURCE_URL="git@github.com:$MM_REPOSITORY_OWNER/$MM_REPOSITORY_NAME.git"
else
	SOURCE_URL="https://github.com/$MM_REPOSITORY_OWNER/$MM_REPOSITORY_NAME.git"
fi
if [[ ! -z "$MM_SOURCE_URL_OVERRIDE" ]]; then
	SOURCE_URL="$MM_SOURCE_URL_OVERRIDE"
fi
if [[ ! -d $MM_REPOSITORY_NAME.git ]]; then
	echo "Cloning $SOURCE_URL to $(pwd)/$MM_REPOSITORY_NAME.git"
	git clone --quiet --mirror $SOURCE_URL $MM_REPOSITORY_NAME.git
	cd $MM_REPOSITORY_NAME.git
	git remote set-url --push origin $MM_TARGET_URL
else
	cd $MM_REPOSITORY_NAME.git
	git fetch --quiet -p origin
fi
if [[ ! -z "$MM_TARGET_KEY_PATH" ]]; then
	export GIT_SSH_COMMAND="ssh -o StrictHostKeyChecking=no -i $MM_TARGET_KEY_PATH"
else
	unset GIT_SSH_COMMAND
fi
git push --quiet --mirror
echo "Mirroring from $(git remote get-url origin) to $(git remote get-url --push origin) complete"
exit 0
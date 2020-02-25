#!/bin/sh

foname=cygo
rm -rf ./$foname
git clone . $foname
du -hs $foname

cd $foname

skip_commit()
{
	  shift;
	  while [ -n "$1" ];
	  do
		    shift;
		    map "$1";
		    shift;
	  done;
}

# delete imported subtree
git filter-branch -f --commit-filter '
	if [ "$GIT_AUTHOR_EMAIL" = "p@pwaller.net" ];
	then
		skip_commit "$@";
	else
		git commit-tree "$@";
	fi' HEAD

# delete un
git filter-branch -f --commit-filter '
  has0=$(git show $GIT_COMMIT --name-status | egrep ^[AMD] | grep noro)
  has1=$(git show $GIT_COMMIT --name-status | egrep ^[AMD] | grep corona)
  has2=$(git show $GIT_COMMIT --name-status | egrep ^[AMD] | grep readme.md)
  has3=$(git show $GIT_COMMIT --name-status | egrep ^[AMD] | grep docs/debug.md)
  if [ "$has0" != "" ] || [ "$has1" != "" ] || [ "$has2" != "" ] || [ "$has3" != "" ]; then
      git commit-tree "$@";
  else
      skip_commit "$@";
  fi
	' HEAD

git filter-branch -f --tree-filter "rm -fr byir bysrc include misc scripts docs/pass.md Makefile" HEAD

git gc
git log --oneline | wc -l

cd -
du -hs $foname


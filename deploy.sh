#!/bin/bash -xe
GIT_REV=`git log --pretty=format:'%h' -n 1`
BUILD_DATE=`date +%Y-%m-%d:%H:%M:%S`
COMPILEINFO=`echo +$GIT_REV+$BUILD_DATE | tr -d '"'`

VERSION=`cat main.go| grep "const AppVersion" |cut -f4 -d " " | tr -d '"'`

#Remember to set GOROOT accordingly with your installation

export GOPATH=$PWD

rm -rf distrib/

declare -a target_folders=("linux64" "linux32" "linuxarm" "osx" "windows")

mkdir distrib

for folder in "${target_folders[@]}"
do
   mkdir -p distrib/$folder/bin/
   go build -o distrib/$folder/bin/arduinoOTA -ldflags "-X main.compileInfo=$COMPILEINFO" main.go

done

#Fix windows binary extension
mv distrib/windows/bin/arduinoOTA distrib/windows/bin/arduinoOTA.exe

cd distrib

for folder in "${target_folders[@]}"
do
   cd $folder
   if [[ $folder == "windows" ]]; then
	zip -r ../arduinoOTA-$VERSION-windows.zip bin/
   else
	tar cjf ../arduinoOTA-$VERSION-$folder.tar.bz2 bin/
   fi
   cd ..
done

echo =======
ls -la arduinoOTA*
echo =======
sha256sum arduinoOTA*
echo =======
shasum arduinoOTA*


#!/bin/bash -xe
GIT_REV=`git log --pretty=format:'%h' -n 1`
BUILD_DATE=`date +%Y-%m-%d:%H:%M:%S`
COMPILEINFO=`echo +$GIT_REV+$BUILD_DATE | tr -d '"'`

VERSION=`cat main.go| grep "const AppVersion" |cut -f4 -d " " | tr -d '"'`

#Remember to set GOROOT accordingly with your installation

export GOPATH=$PWD

rm -rf distrib/

declare -a target_folders=("linux_amd64" "linux_386" "linux_arm" "darwin_amd64" "windows_386")

mkdir distrib

for folder in "${target_folders[@]}"
do
   IFS=_ read -a fields <<< $folder
   mkdir -p distrib/$folder/bin/
   GOOS=${fields[0]} GOARCH=${fields[1]} go build -o distrib/$folder/bin/arduinoOTA -ldflags "-X main.compileInfo=$COMPILEINFO" main.go

done

#Fix windows binary extension
mv distrib/windows_386/bin/arduinoOTA distrib/windows_386/bin/arduinoOTA.exe

cd distrib

for folder in "${target_folders[@]}"
do
   cd $folder
   if [[ $folder == "windows*" ]]; then
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


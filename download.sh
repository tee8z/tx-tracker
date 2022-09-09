#!/bin/bash

helpFunction()
{
   echo ""
   echo "Usage: $0 -v version "
   echo -e "\t-v Version of release to install"
   exit 1 # Exit script after printing help
}

while getopts "v:i" opt
do
   case "$opt" in
      v ) version="$OPTARG" ;;
      i ) initial_install=true;;
      ? ) helpFunction ;; # Print helpFunction in case parameter is non-existent
   esac
done

# Print helpFunction in case parameters are empty
if [ -z "$version" ]
then
   echo "the version parameter is missing";
   helpFunction
fi


wget "https://github.com/tee8z/tx-tracker/releases/download/${version}/tx-tracker-${version}-linux-amd64.tar.gz"
wget "https://github.com/tee8z/tx-tracker/releases/download/${version}/tx-tracker-${version}-linux-amd64.tar.gz.md5"
echo "$(cat tx-tracker-${version}-linux-amd64.tar.gz.md5)" tx-tracker-${version}-linux-amd64.tar.gz |  md5sum -c -
mkdir service
tar -xvzf "tx-tracker-${version}-linux-amd64.tar.gz" -C service
chmod -R 777 service/tx-tracker
rm tx-tracker-${version}-linux-amd64.tar.gz.md5
rm tx-tracker-${version}-linux-amd64.tar.gz

if $initial_install
then 
   cp default.env service/.env
fi

#! /bin/sh

echo Deleting cattlestore app from Marathon

http DELETE $(docker-machine ip default):8080/v2/apps/cattlestore


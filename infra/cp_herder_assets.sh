#! /bin/sh

docker cp ../assets/herder.css infra_herder_1:/assets/herder.css
docker cp ../assets/app.js infra_herder_1:/assets/app.js
docker cp ../index.html infra_herder_1:/index.html


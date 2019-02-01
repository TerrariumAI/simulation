echo "[Building docker image]"
docker build -t simulation .

echo "[Publishing docker image]"
docker login
docker tag simulation olamai/simulation:0.0.1
docker push olamai/simulation:0.0.1

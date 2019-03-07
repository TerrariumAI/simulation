echo "[Building server executable]"
./build.sh

echo "[Building docker image]"
docker build --tag olamai/simulation:0.0.1 .

echo "[Publishing docker image]"
docker login
docker push olamai/simulation:0.0.1
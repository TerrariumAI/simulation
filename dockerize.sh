echo "[Building docker image]"
docker build -t datacom .

echo "[Publishing docker image]"
docker login
docker tag datacom olamai/datacom:0.0.1
docker push olamai/datacom:0.0.1
